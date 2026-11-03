# `localhost` pages were slow: the Windows hosts file had localhost commented out

**Branch:** `feature/mvp-phases-3-7`
**Reported:** maintainer — "The Go server is irritatingly very slow on localhost, pages are taking time to load, even the Next.js frontend compiling takes long."
**Status:** Fixed (machine configuration, not repository code).

---

## Symptom

Every request to `http://localhost:…` was slow, in every tool: the browser, the
Next.js dev server, `curl`, PowerShell and Node. The API itself was healthy —
the two servers were running and answering `200`.

## Reproduction

Time the same endpoint by name and by address, from the same shell:

```powershell
# Node (the path Next.js SSR uses)
node -e "const t=async u=>{const s=Date.now();await (await fetch(u)).text();console.log(u,(Date.now()-s)+'ms')};(async()=>{await t('http://localhost:8080/api/health');await t('http://127.0.0.1:8080/api/health')})()"
```

Before the fix:

| Target | First hit | Warm |
|---|---|---|
| `http://localhost:8080/api/health` | **418 ms** | 45 ms |
| `http://127.0.0.1:8080/api/health` | 14 ms | 16 ms |

`curl` showed the same shape (648 ms vs 8 ms), and a cold name lookup was the
whole cost:

```
Resolve-DnsName localhost   ->  2,282 ms   (::1, 127.0.0.1)
Resolve-DnsName localhost   ->      4 ms   (cached)
```

A TCP connect was ~2 ms, so the delay was name resolution, not the server.

## Root cause

`C:\Windows\System32\drivers\etc\hosts` had its loopback mappings **commented
out** (these are the stock Windows lines):

```
# localhost name resolution is handled within DNS itself.
#	127.0.0.1       localhost
#	::1             localhost
```

(The file also carries Docker Desktop's `host.docker.internal` entries, which is
what makes the file non-stock in the first place.)

With no local mapping, the Windows resolver sent a `localhost` query to the
configured DNS server — here 8.8.8.8 / 8.8.4.4 over Wi-Fi — and only fell back
to the loopback addresses after that round-trip. Every process that resolved the
name paid it once per process (and periodically, as the DNS cache aged).

That single cost surfaced as:

- the API "being slow": each new client/SSR connection stalled on the lookup;
- the Next.js dev server "compiling slowly": the browser's HMR and RSC
  connections to `localhost:3000`, plus any server-side fetch to the API,
  inherited the same lookup;
- `curl` and PowerShell looking slow against an endpoint that answered in 8 ms.

## Fix

Restore the standard loopback mappings in the hosts file (needs an elevated
shell; the change is machine-wide, not part of the repository):

```powershell
# Run in an Administrator PowerShell
$p = "$env:SystemRoot\System32\drivers\etc\hosts"
Add-Content -Path $p -Value "127.0.0.1`tlocalhost`n::1`tlocalhost"
ipconfig /flushdns
```

## Verification

After the change, with the API still running:

| Target | Result |
|---|---|
| `curl http://localhost:8080/api/health` | 8–14 ms, `200` |
| `curl http://localhost:3000/elections` | 155 ms, `200` |
| `ping localhost` | resolves to `::1` immediately |
| Node `fetch` to `http://localhost:8080` | 13 ms warm |

No application files were changed, because the cause was outside the repository.

## Notes / limitations

- `Resolve-DnsName localhost` still reports a few hundred ms because that cmdlet
  performs a real DNS query rather than reading the hosts file; it is not a
  useful probe here. Use a connection timing (above) instead.
- If the hosts file is ever reset (some VPN/security tools rewrite it), the
  symptom returns; the fix is the same.
- Application-level workarounds such as pointing `NEXT_PUBLIC_API_URL` at
  `127.0.0.1` were deliberately **not** taken: the browser page would still be
  served from `localhost`, and a `localhost` page calling a `127.0.0.1` API is a
  cross-site request, so the API's `SameSite=Lax` session cookie would not be
  sent and sign-in would break. The hosts entry keeps the page and the API on
  the same site.
