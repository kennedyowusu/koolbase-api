package invitations

import (
	"fmt"
	"time"
)

func currentYear() string {
	return fmt.Sprintf("%d", time.Now().Year())
}

func inviteEmailHTML(orgName, inviteeEmail, role, acceptURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0a0c10; color: #f1f5f9; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; }
</style>
</head>
<body style="background-color:#0a0c10;margin:0;padding:48px 16px;">

<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center" style="padding-bottom:24px;">
    <table border="0" cellpadding="0" cellspacing="0">
      <tr>
        <td style="padding-right:10px;">
          <svg fill="none" height="32" width="32" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
            <path d="M44 4H30.6666V17.3334H17.3334V30.6666H4V44H44V4Z" fill="#2b8cee"/>
          </svg>
        </td>
        <td style="font-size:22px;font-weight:700;color:#ffffff;">Koolbase</td>
      </tr>
    </table>
  </td></tr>
</table>

<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center">
    <table border="0" cellpadding="0" cellspacing="0" width="560" style="background-color:#111418;border:1px solid #1e2531;border-radius:12px;overflow:hidden;">
      <tr><td align="center" style="padding:40px 40px 32px 40px;">

        <div style="background-color:rgba(43,140,238,0.1);border-radius:50%%;width:64px;height:64px;display:inline-block;text-align:center;line-height:64px;font-size:32px;margin-bottom:24px;">🤝</div>

        <h1 style="font-size:26px;font-weight:700;color:#ffffff;margin:0 0 12px 0;">You're invited to join Koolbase</h1>
        <p style="font-size:15px;color:#94a3b8;line-height:1.7;margin:0 0 24px 0;">
          You've been invited to join <strong style="color:#ffffff;">%s</strong> on Koolbase as a <strong style="color:#2b8cee;">%s</strong>.
        </p>

        <a href="%s" style="display:inline-block;background-color:#2b8cee;color:#ffffff;font-size:16px;font-weight:600;padding:14px 36px;border-radius:8px;margin-bottom:32px;">
          Accept Invitation
        </a>

        <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-bottom:24px;">
          <tr><td style="border-top:1px solid #1e2531;padding-top:24px;">
            <p style="font-size:13px;color:#64748b;text-align:center;margin:0 0 10px 0;">Or copy and paste this link into your browser:</p>
            <div style="background-color:#0d1117;border:1px solid #1e2531;border-radius:6px;padding:12px;word-break:break-all;text-align:center;">
              <a href="%s" style="font-family:'Courier New',monospace;font-size:11px;color:#2b8cee;">%s</a>
            </div>
          </td></tr>
        </table>

        <table border="0" cellpadding="0" cellspacing="0" width="100%%">
          <tr><td style="background-color:rgba(245,158,11,0.05);border:1px solid rgba(245,158,11,0.2);border-radius:8px;padding:14px 16px;">
            <p style="font-size:12px;color:#94a3b8;margin:0;line-height:1.6;">
              <strong style="color:#f59e0b;">Note:</strong> This invitation expires in 48 hours.
              If you did not expect this invitation, you can safely ignore this email.
            </p>
          </td></tr>
        </table>

      </td></tr>
    </table>
  </td></tr>
</table>

<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:32px;">
  <tr><td align="center" style="font-size:12px;color:#475569;">
    <div style="margin-bottom:12px;">
      <a href="https://docs.koolbase.com" style="color:#475569;margin:0 10px;">Docs</a> •
      <a href="mailto:techfinityedge@gmail.com" style="color:#475569;margin:0 10px;">Support</a> •
      <a href="https://koolbase.com/terms" style="color:#475569;margin:0 10px;">Terms</a> •
      <a href="https://koolbase.com/privacy" style="color:#475569;margin:0 10px;">Privacy</a>
    </div>
    <p style="margin:0;color:#334155;">© %s Koolbase Inc. All rights reserved.</p>
  </td></tr>
</table>

</body>
</html>`, orgName, role, acceptURL, acceptURL, acceptURL, currentYear())
}

func revokeInviteEmailHTML(orgName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0a0a0a; color: #ededed; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; }
</style>
</head>
<body style="background-color:#0a0a0a;margin:0;padding:48px 24px;">

<!-- Header -->
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center" style="padding-bottom:40px;">
    <div style="display:inline-block;width:48px;height:48px;background-color:#ffffff;border-radius:10px;text-align:center;line-height:48px;font-size:20px;font-weight:700;color:#0a0a0a;margin-bottom:12px;">K</div>
    <p style="font-size:11px;font-weight:500;letter-spacing:0.2em;text-transform:uppercase;color:#ededed;font-family:'Courier New',monospace;margin:0;">Koolbase</p>
  </td></tr>
</table>

<!-- Card -->
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center">
    <table border="0" cellpadding="0" cellspacing="0" width="560" style="background-color:#111111;border:1px solid #2e2e2e;border-radius:12px;overflow:hidden;">
      <tr><td style="padding:40px;">

        <h2 style="font-size:24px;font-weight:600;color:#ffffff;margin:0 0 24px 0;">Invitation Revoked</h2>

        <p style="font-size:15px;color:#a1a1a1;line-height:1.7;margin:0 0 16px 0;">
          The invitation for you to join <strong style="color:#ffffff;">%s</strong> on Koolbase has been cancelled by an administrator.
        </p>

        <p style="font-size:15px;color:#a1a1a1;line-height:1.7;margin:0 0 24px 0;">
          This means you no longer have access to the pending workspace or its resources. Any access tokens associated with this invitation have been invalidated.
        </p>

        <!-- Note -->
        <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-bottom:32px;">
          <tr><td style="background-color:#1a1a1a;border-left:2px solid #3e3e3e;padding:16px 20px;">
            <p style="font-size:13px;color:#a1a1a1;font-style:italic;margin:0;line-height:1.6;">
              If you believe this is a mistake, please contact your team lead or the person who originally sent the invitation.
            </p>
          </td></tr>
        </table>

        <!-- CTAs -->
        <table border="0" cellpadding="0" cellspacing="0">
          <tr>
            <td style="padding-right:12px;">
              <a href="https://koolbase.com" style="display:inline-block;padding:12px 24px;background-color:#ededed;color:#0a0a0a;font-size:14px;font-weight:600;border-radius:6px;">
                Go to Koolbase
              </a>
            </td>
            <td>
              <a href="mailto:techfinityedge@gmail.com" style="display:inline-block;padding:12px 24px;border:1px solid #2e2e2e;color:#ededed;font-size:14px;font-weight:500;border-radius:6px;">
                Contact Support
              </a>
            </td>
          </tr>
        </table>

      </td></tr>
    </table>
  </td></tr>
</table>

<!-- Footer -->
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:40px;">
  <tr><td align="center">
    <div style="margin-bottom:16px;">
      <a href="https://docs.koolbase.com" style="color:#707070;font-size:11px;font-family:'Courier New',monospace;text-transform:uppercase;letter-spacing:0.15em;margin:0 12px;">Docs</a>
      <a href="mailto:techfinityedge@gmail.com" style="color:#707070;font-size:11px;font-family:'Courier New',monospace;text-transform:uppercase;letter-spacing:0.15em;margin:0 12px;">Support</a>
      <a href="https://koolbase.com/terms" style="color:#707070;font-size:11px;font-family:'Courier New',monospace;text-transform:uppercase;letter-spacing:0.15em;margin:0 12px;">Terms</a>
      <a href="https://koolbase.com/privacy" style="color:#707070;font-size:11px;font-family:'Courier New',monospace;text-transform:uppercase;letter-spacing:0.15em;margin:0 12px;">Privacy</a>
    </div>
    <p style="font-size:10px;color:#525252;letter-spacing:0.2em;text-transform:uppercase;font-family:'Courier New',monospace;margin:0;">© %s Koolbase Inc. All rights reserved.</p>
  </td></tr>
</table>

</body>
</html>`, orgName, currentYear())
}
