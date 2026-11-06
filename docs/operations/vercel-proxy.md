# Hosting the web app on Vercel, API elsewhere (proxy setup)

Status: implemented, not verified end-to-end against a live Vercel deployment.

## Why a proxy is needed

A Vercel deployment is served over HTTPS. Calling a plain-HTTP API from that
page is blocked by the browser as mixed content, so an address like
`http://194.36.88.77:8080` cannot be used as `NEXT_PUBLIC_API_URL` directly.

Even with HTTPS on the API, `*.vercel.app` is a public-suffix domain: it shares
no registrable domain with any API host, so every API host is a different
*site* to the browser. The API's session cookies are `SameSite=Lax`
(`apps/api/internal/services/auth.go`), and browsers drop those on cross-site
requests — sign-in returns 200, then every request after it is 401. The API
logs a warning about this arrangement at boot
(`warnCrossSiteAuth`, `apps/api/internal/config/config.go`).

## Setup

`apps/web/next.config.ts` proxies `/api/*` and `/files/*` to `API_PROXY_TARGET`
when that variable is set. The browser then only ever talks to the Vercel
origin: requests are same-origin, CORS does not apply, and the API's cookies
stay first-party.

In the Vercel project, under **Settings → Environment Variables** (Production):

| Variable | Value | Why |
|---|---|---|
| `API_PROXY_TARGET` | `http://194.36.88.77:8080` | The API the rewrite forwards to, server-side. No slash at the end. |
| `NEXT_PUBLIC_API_URL` | *(unset)* | Empty means "this origin", so requests carry no hostname. Delete it if it was set to the production origin. |

Set `API_PROXY_TARGET` for **Production and Preview**. Preview hostnames cannot
be known when a build starts, so an absolute `NEXT_PUBLIC_API_URL` is always
wrong for previews; the relative default is what makes them work unchanged.

Then **redeploy**: the rewrite is read at build time, so setting the variable
alone does not update the running deployment. A production build with neither
variable prints a warning, because its `/api` calls would 404 against the web
app itself.

On the API, add the web origin to `CORS_ORIGINS` (comma separated, then restart)
or to the stored `cors.origins` setting. The proxy forwards the browser's
`Origin` header, so without it the API logs `CORS: refused ...` once per origin
even though the proxied requests still work.

`APP_URL` on the API should be the address the browser knows it by
(`http://194.36.88.77:8080` here); stored file links are rewritten to the API
base by `apps/web/lib/api.ts` either way.

## Limits of this arrangement

- **WebSocket.** Vercel does not pass WebSocket upgrades, so `/api/ws` fails the
  handshake twice and `apps/web/lib/realtime.ts` switches to SSE
  (`/api/events`) for the rest of the session. Realtime depends on the rewrite
  passing that stream through; if it does not, realtime is unavailable but the
  rest of the app works.
- **Upload size.** Proxied requests pass through Vercel's request limits, which
  are smaller than calling the API directly. Keep upload limits modest.
- **API load.** The API sees Vercel's egress address, not the visitor's, so
  per-IP rate limiting on the API is less meaningful than in a direct setup.

## Local development

Leave `API_PROXY_TARGET` unset. `.env` sets
`NEXT_PUBLIC_API_URL=http://localhost:8080` and the browser calls the API
directly; both are on `localhost`, so there is no mixed content and no
cross-site cookie problem.
