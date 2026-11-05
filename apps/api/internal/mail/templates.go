package mail

// The six emails, as fragments. Layout is the shell they are rendered into.
//
// Fragments rather than whole documents, because six copies of one shell is how
// they drifted apart last time: three different footers and two different greys
// between them.
var EmailTemplates = map[string]string{
	"welcome":            welcomeTemplate,
	"password-reset":     passwordResetTemplate,
	"email-verification": emailVerificationTemplate,
	"notification":       notificationTemplate,
	"two-factor-code":    twoFactorCodeTemplate,
	"magic-link":         magicLinkTemplate,
}

// Layout wraps every fragment.
//
// Tables and style attributes, which is not a stylistic choice. Outlook on
// Windows renders mail through Word, which ignores most of a <style> block and
// all of flexbox; Gmail drops <style> when it clips a long message or when
// somebody forwards it. Anything that has to survive goes on the element.
//
// The style strings the fragments use ({{.H1}}, {{.P}} and the rest) are
// supplied by the renderer, so there is one definition of what a heading looks
// like rather than one per template.
const Layout = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<meta name="x-apple-disable-message-reformatting" />
<meta name="color-scheme" content="light" />
<meta name="supported-color-schemes" content="light" />
<title>{{.Subject}}</title>
</head>
<body style="margin:0;padding:0;background-color:#F3F6F4;">
<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;font-size:1px;line-height:1px;color:#F3F6F4;">{{.Preheader}}&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;&#8203;</div>

<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F3F6F4;">
<tr><td align="center" style="padding:24px 12px 32px;">

<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:580px;width:100%;">

  <!-- ── Header band: #0A0F0C matching the site's admin sidebar ── -->
  <tr>
    <td style="background-color:#0A0F0C;padding:0;">
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <!-- Logo mark: a square box (the .mark pattern from the kit) + wordmark -->
        <td style="padding:14px 20px;" width="1">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <!-- .mark: --ink bg, 28×28, grid place-items:center -->
            <td style="padding-right:10px;vertical-align:middle;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="background-color:#0A0F0C;border:1.5px solid #3E4A44;width:28px;height:28px;">
              <tr><td align="center" valign="middle" style="width:28px;height:28px;">
                <!-- ballot icon: path from i-ballot in the kit -->
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#C9D2CD" stroke-width="1.75" stroke-linecap="square" stroke-linejoin="miter" xmlns="http://www.w3.org/2000/svg"><path d="M3 13h18v8H3zM7 13V3h10v10M10 8h4"/></svg>
              </td></tr>
              </table>
            </td>
            <!-- wordmark: "camp" light + "Vote" bold, matching .brand in the kit -->
            <td style="vertical-align:middle;white-space:nowrap;">
              <span style="font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:16px;font-weight:400;color:#8D9B94;letter-spacing:-0.01em;">camp</span><span style="font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:16px;font-weight:700;color:#F2F7F4;letter-spacing:-0.01em;">Vote</span>
            </td>
          </tr>
          </table>
        </td>
        <!-- ACCOUNT label — right side, mono uppercase, like the nav items -->
        <td align="right" style="padding:14px 20px;font-family:'IBM Plex Mono','SFMono-Regular',Consolas,monospace;font-size:11px;font-weight:500;letter-spacing:0.08em;text-transform:uppercase;color:#8D9B94;white-space:nowrap;">
          ACCOUNT
        </td>
      </tr>
      </table>
    </td>
  </tr>

  <!-- ── Card: white, 1.5px #0A0F0C border (strong), 0px radius ── -->
  <tr>
    <td style="background-color:#FFFFFF;border-left:1.5px solid #0A0F0C;border-right:1.5px solid #0A0F0C;padding:32px 32px 28px;">
      {{.Content}}
    </td>
  </tr>

  <!-- ── Footer: sunken #F3F6F4, 1px #D3D9D6 hairline border ── -->
  <tr>
    <td style="background-color:#F3F6F4;border:1.5px solid #0A0F0C;border-top:1px solid #D3D9D6;padding:14px 20px;">
      <p style="margin:0;font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:12px;line-height:1.7;color:#5B6761;">
        Sent to {{.RecipientEmail}} because you have a {{.AppName}} account. This is an automated message and replies are not monitored.{{if .SupportEmail}} Need help? <a href="mailto:{{.SupportEmail}}" style="color:#12783F;text-decoration:none;">{{.SupportEmail}}</a>{{end}}<br>
        Times are in EAT (UTC+3). {{.AppName}} will never ask for your password by email.
      </p>
    </td>
  </tr>

</table>
</td></tr>
</table>
</body>
</html>`

// Fragments: the inside of the card, and nothing else.
//
// Divs with explicit margins rather than h1/p, because Outlook adds margins of
// its own to block elements that no stylesheet can take back.

const welcomeTemplate = `<div style="{{.H1}}">Welcome to {{.AppName}}</div>
<div style="{{.P}}">Hi {{.Name}}, your account is ready. There is nothing else to set up.</div>
{{if .ActionURL}}<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 20px;"><tr><td align="center" style="{{.BtnCell}}"><a href="{{.ActionURL}}" style="{{.BtnLink}}">{{.ActionText}}</a></td></tr></table>{{end}}
<div style="{{.PLast}}">If you did not create this account you can ignore this message, and nothing further will happen.</div>`

const passwordResetTemplate = `<div style="{{.H1}}">Reset your password</div>
<div style="{{.P}}">Somebody asked to reset the password for this address. Use the button below to choose a new one.</div>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 20px;"><tr><td align="center" style="{{.BtnCell}}"><a href="{{.ResetURL}}" style="{{.BtnLink}}">Choose a new password</a></td></tr></table>
<div style="{{.PLast}}">The link works once and expires in an hour. If this was not you, nothing has changed and you can ignore this message.</div>`

const emailVerificationTemplate = `
<!-- ── Label: mono uppercase primary (kit .lbl pattern) ── -->
<p style="margin:0 0 10px;font-family:'IBM Plex Mono','SFMono-Regular',Consolas,monospace;font-size:11px;font-weight:500;letter-spacing:0.08em;text-transform:uppercase;color:#12783F;">Verify your email</p>

<!-- ── Heading (kit h1: 700, tight letter-spacing) ── -->
<h1 style="margin:0 0 14px;font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:24px;line-height:1.2;font-weight:700;letter-spacing:-0.02em;color:#0A0F0C;">Your verification code</h1>

<!-- ── Body copy ── -->
<p style="margin:0 0 24px;font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:15px;line-height:1.65;color:#38423D;">{{if .FirstName}}Hi {{.FirstName}}, enter{{else}}Enter{{end}} this code in {{.AppName}} to finish signing in.</p>

<!-- ── OTP digit boxes (kit .otp style: 48×56 each, mono, 1.5px strong border) ── -->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 8px;">
<tr>
  <td style="background-color:#FFFFFF;border:1.5px solid #0A0F0C;padding:20px 16px 14px;">
    <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0 auto;">
    <tr>
      <!-- Digit cells: 48×56, mono, 1.5px #0A0F0C border, 0px radius -->
      <!-- Gap of 16px after cell 3 (two groups: 3 + 3) -->
      {{range $i, $ch := .Digits}}<td style="padding:{{if eq $i 3}}0 4px 0 20px{{else}}0 4px{{end}};">
        <table role="presentation" cellpadding="0" cellspacing="0" border="0">
        <tr><td align="center" valign="middle" style="width:44px;height:52px;border:1.5px solid #0A0F0C;font-family:'IBM Plex Mono','SFMono-Regular',Consolas,monospace;font-size:26px;font-weight:500;color:#0A0F0C;text-align:center;">{{$ch}}</td></tr>
        </table>
      </td>{{end}}
    </tr>
    </table>
    <!-- "Expires in N minutes · single use" — mono 12px --ink-3 -->
    <p style="margin:12px 0 0;text-align:center;font-family:'IBM Plex Mono','SFMono-Regular',Consolas,monospace;font-size:12px;color:#5B6761;letter-spacing:0.04em;">Expires in {{.Minutes}} minutes &middot; single use</p>
  </td>
</tr>
</table>

<!-- ── Copy bar: large selectable code for easy copying ── -->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 20px;">
<tr>
  <td style="background-color:#F3F6F4;border:1.5px solid #0A0F0C;padding:12px 16px;">
    <p style="margin:0 0 4px;font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:11px;font-weight:600;letter-spacing:0.08em;text-transform:uppercase;color:#5B6761;">Select to copy code</p>
    <p style="margin:0;font-family:'IBM Plex Mono','SFMono-Regular',Consolas,monospace;font-size:22px;font-weight:700;letter-spacing:0.15em;color:#0A0F0C;line-height:1.3;">{{.CodeRaw}}</p>
  </td>
</tr>
</table>

<!-- ── Security banner: .banner.ok — #EDF7F1 bg, 1px #AEDBC0 border, shield icon ── -->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0;">
<tr>
  <td style="background-color:#EDF7F1;border:1px solid #AEDBC0;padding:12px 14px;">
    <table role="presentation" cellpadding="0" cellspacing="0" border="0">
    <tr>
      <!-- shield icon from the kit: i-shield -->
      <td style="padding-right:10px;vertical-align:top;padding-top:2px;">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#12783F" stroke-width="1.75" stroke-linecap="square" stroke-linejoin="miter" xmlns="http://www.w3.org/2000/svg"><path d="M12 2 4 5v7c0 5 3.5 8.5 8 10 4.5-1.5 8-5 8-10V5zM9 12l2 2 4-4"/></svg>
      </td>
      <td>
        <p style="margin:0 0 3px;font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:14px;font-weight:700;color:#0A4A27;">Never share this code</p>
        <p style="margin:0;font-family:'Space Grotesk','Segoe UI',system-ui,sans-serif;font-size:13px;line-height:1.55;color:#12783F;">{{.AppName}} staff will never ask for it. If you did not request it, you can ignore this email.</p>
      </td>
    </tr>
    </table>
  </td>
</tr>
</table>`

const notificationTemplate = `<div style="{{.H1}}">{{.Title}}</div>
<div style="{{.P}}">{{.Message}}</div>
{{if .ActionURL}}<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0;"><tr><td align="center" style="{{.BtnCell}}"><a href="{{.ActionURL}}" style="{{.BtnLink}}">{{.ActionText}}</a></td></tr></table>{{end}}`

const magicLinkTemplate = `<div style="{{.H1}}">Your sign-in link</div>
<div style="{{.P}}">Use the button below to sign in. There is no password to type.</div>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 20px;"><tr><td align="center" style="{{.BtnCell}}"><a href="{{.MagicURL}}" style="{{.BtnLink}}">Sign in</a></td></tr></table>
<div style="{{.PLast}}">The link lasts {{.Minutes}} minutes and works once. If you did not ask for it, somebody knows your email address, which is not a secret: they cannot sign in without this mailbox.</div>`

const twoFactorCodeTemplate = `<div style="{{.H1}}">Your sign-in code</div>
<div style="{{.P}}">Type this into the page you already have open.</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 18px;">
<tr><td align="center" style="background-color:{{.Canvas}};border:1px solid {{.Border}};border-radius:12px;padding:20px 12px;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:32px;font-weight:700;letter-spacing:10px;color:{{.Ink}};">{{.Code}}</td></tr>
</table>
<div style="{{.PLast}}">It expires in {{.Minutes}} minutes and works once. If you were not signing in, somebody has your password: change it. This code alone lets nobody in.</div>`
