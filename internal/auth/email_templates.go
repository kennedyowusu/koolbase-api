package auth

import "fmt"

func verificationEmailHTML(verifyURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<meta content="IE=edge" http-equiv="X-UA-Compatible"/>
<title>Verify your Koolbase account</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0B0E14; color: #ffffff; margin: 0; padding: 0; width: 100%%; }
  table { border-collapse: collapse; }
  a { text-decoration: none; color: #2b8cee; }
</style>
</head>
<body style="background-color:#0B0E14;margin:0;padding:0;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="background-color:#0B0E14;">
  <tr>
    <td align="center" style="padding:48px 16px;">
      <table border="0" cellpadding="0" cellspacing="0" width="520" style="background-color:#161B22;border:1px solid #30363D;border-radius:12px;overflow:hidden;">
        <tr>
          <td align="center" style="padding:40px 32px 24px 32px;">
            <svg fill="none" height="40" width="40" viewBox="0 0 40 40" xmlns="http://www.w3.org/2000/svg" style="display:block;margin:0 auto 16px auto;">
              <rect fill="#2B8CEE" height="40" rx="8" width="40"/>
              <path d="M12 12H28V16L20 24L12 16V12Z" fill="white"/>
              <rect fill="white" fill-opacity="0.6" height="4" width="8" x="16" y="24"/>
            </svg>
            <div style="font-size:20px;font-weight:700;letter-spacing:-0.5px;margin-bottom:24px;">Koolbase</div>
            <h1 style="font-size:24px;font-weight:600;color:#ffffff;margin:0 0 16px 0;line-height:1.3;">Verify your email</h1>
            <p style="font-size:16px;color:#9CA3AF;line-height:1.6;margin:0;">
              Please confirm your email address to activate your Koolbase account and start managing your feature flags.
            </p>
          </td>
        </tr>
        <tr>
          <td align="center" style="padding:0 32px 40px 32px;">
            <a href="%s" style="display:inline-block;background-color:#2B8CEE;color:#ffffff;font-size:16px;font-weight:600;padding:14px 32px;border-radius:8px;margin-bottom:32px;">
              Verify Email
            </a>
            <p style="font-size:13px;color:#6B7280;margin:0 0 8px 0;">Or copy and paste this URL into your browser:</p>
            <div style="background-color:#0B0E14;border:1px solid #30363D;border-radius:6px;padding:12px;word-break:break-all;">
              <a href="%s" style="font-family:'Courier New',monospace;font-size:12px;color:#2B8CEE;">%s</a>
            </div>
            <p style="font-size:12px;color:#6B7280;font-style:italic;margin:24px 0 0 0;">
              This link expires in 24 hours. If you did not sign up for Koolbase, you can safely ignore this email.
            </p>
          </td>
        </tr>
      </table>
      <table border="0" cellpadding="0" cellspacing="0" width="520" style="margin-top:32px;">
        <tr>
          <td align="center" style="font-size:12px;color:#6B7280;">
            <div style="margin-bottom:12px;">
              <a href="https://docs.koolbase.com" style="color:#6B7280;margin:0 8px;">Docs</a>
              <span style="color:#374151;">•</span>
              <a href="https://koolbase.com/support" style="color:#6B7280;margin:0 8px;">Support</a>
              <span style="color:#374151;">•</span>
              <a href="https://koolbase.com/terms" style="color:#6B7280;margin:0 8px;">Terms</a>
              <span style="color:#374151;">•</span>
              <a href="https://koolbase.com/privacy" style="color:#6B7280;margin:0 8px;">Privacy</a>
            </div>
            <p style="margin:0;">© 2025 Koolbase, Inc. All rights reserved.</p>
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>
</body>
</html>`, verifyURL, verifyURL, verifyURL)
}

func passwordResetEmailHTML(resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<meta content="IE=edge" http-equiv="X-UA-Compatible"/>
<title>Reset your password - Koolbase</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0a0c10; color: #f1f5f9; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; color: #2b8cee; }
</style>
</head>
<body style="background-color:#0a0c10;margin:0;padding:32px 16px;">

  <!-- Logo -->
  <table border="0" cellpadding="0" cellspacing="0" width="100%%">
    <tr>
      <td align="center" style="padding-bottom:24px;">
        <table border="0" cellpadding="0" cellspacing="0">
          <tr>
            <td style="padding-right:10px;">
              <svg fill="none" height="32" width="32" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
                <path d="M44 4H30.6666V17.3334H17.3334V30.6666H4V44H44V4Z" fill="#2b8cee"/>
              </svg>
            </td>
            <td style="font-size:22px;font-weight:700;color:#ffffff;letter-spacing:-0.5px;">Koolbase</td>
          </tr>
        </table>
      </td>
    </tr>
  </table>

  <!-- Main Card -->
  <table border="0" cellpadding="0" cellspacing="0" width="100%%">
    <tr>
      <td align="center">
        <table border="0" cellpadding="0" cellspacing="0" width="560" style="background-color:#111418;border:1px solid #1e2531;border-radius:12px;overflow:hidden;">
          <tr>
            <td align="center" style="padding:40px 40px 32px 40px;">

              <!-- Icon -->
              <div style="background-color:rgba(43,140,238,0.1);border-radius:50%%;width:64px;height:64px;display:inline-block;text-align:center;line-height:64px;margin-bottom:24px;">
                <span style="font-size:32px;">🔑</span>
              </div>

              <h1 style="font-size:26px;font-weight:700;color:#ffffff;margin:0 0 12px 0;line-height:1.3;">Reset your password</h1>
              <p style="font-size:15px;color:#94a3b8;line-height:1.7;margin:0 0 32px 0;">
                We received a request to reset your password for your Koolbase account. Click the button below to choose a new one.
              </p>

              <!-- CTA Button -->
              <a href="%s" style="display:inline-block;background-color:#2b8cee;color:#ffffff;font-size:16px;font-weight:600;padding:14px 36px;border-radius:8px;margin-bottom:32px;">
                Reset Password
              </a>

              <!-- Divider -->
              <table border="0" cellpadding="0" cellspacing="0" width="100%%">
                <tr>
                  <td style="border-top:1px solid #1e2531;padding-top:28px;">
                    <p style="font-size:13px;color:#64748b;text-align:center;margin:0 0 10px 0;">
                      If the button doesn't work, copy and paste this link into your browser:
                    </p>
                    <div style="background-color:#0d1117;border:1px solid #1e2531;border-radius:6px;padding:12px;word-break:break-all;text-align:center;">
                      <a href="%s" style="font-family:'Courier New',monospace;font-size:11px;color:#2b8cee;">%s</a>
                    </div>
                  </td>
                </tr>
              </table>

              <!-- Security Note -->
              <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:24px;">
                <tr>
                  <td style="background-color:rgba(245,158,11,0.05);border:1px solid rgba(245,158,11,0.2);border-radius:8px;padding:14px 16px;">
                    <p style="font-size:12px;color:#94a3b8;margin:0;line-height:1.6;">
                      <strong style="color:#f59e0b;">Security note:</strong> This link will expire in 1 hour. If you didn't request a password reset, you can safely ignore this email. Your account is still secure.
                    </p>
                  </td>
                </tr>
              </table>

            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>

  <!-- Footer -->
  <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:32px;">
    <tr>
      <td align="center" style="font-size:12px;color:#475569;">
        <div style="margin-bottom:12px;">
          <a href="https://docs.koolbase.com" style="color:#475569;margin:0 10px;">Docs</a>
          <span style="color:#334155;">•</span>
          <a href="https://koolbase.com/support" style="color:#475569;margin:0 10px;">Support</a>
          <span style="color:#334155;">•</span>
          <a href="https://koolbase.com/terms" style="color:#475569;margin:0 10px;">Terms</a>
          <span style="color:#334155;">•</span>
          <a href="https://koolbase.com/privacy" style="color:#475569;margin:0 10px;">Privacy</a>
        </div>
        <p style="margin:0;color:#334155;">© 2025 Koolbase Inc. All rights reserved.</p>
      </td>
    </tr>
  </table>

</body>
</html>`, resetURL, resetURL, resetURL)
}
