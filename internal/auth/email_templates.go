package auth

import "fmt"

func verificationEmailHTML(verifyURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0B0E14; color: #ffffff; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; color: #2b8cee; }
</style>
</head>
<body style="background-color:#0B0E14;margin:0;padding:0;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="background-color:#0B0E14;">
  <tr><td align="center" style="padding:48px 16px;">
    <table border="0" cellpadding="0" cellspacing="0" width="520" style="background-color:#161B22;border:1px solid #30363D;border-radius:12px;overflow:hidden;">
      <tr><td align="center" style="padding:40px 32px 24px 32px;">
        <svg fill="none" height="40" width="40" viewBox="0 0 40 40" xmlns="http://www.w3.org/2000/svg" style="display:block;margin:0 auto 16px auto;">
          <rect fill="#2B8CEE" height="40" rx="8" width="40"/>
          <path d="M12 12H28V16L20 24L12 16V12Z" fill="white"/>
          <rect fill="white" fill-opacity="0.6" height="4" width="8" x="16" y="24"/>
        </svg>
        <div style="font-size:20px;font-weight:700;margin-bottom:24px;">Koolbase</div>
        <h1 style="font-size:24px;font-weight:600;color:#ffffff;margin:0 0 16px 0;">Verify your email</h1>
        <p style="font-size:16px;color:#9CA3AF;line-height:1.6;margin:0;">Please confirm your email address to activate your Koolbase account and start managing your feature flags.</p>
      </td></tr>
      <tr><td align="center" style="padding:0 32px 40px 32px;">
        <a href="%s" style="display:inline-block;background-color:#2B8CEE;color:#ffffff;font-size:16px;font-weight:600;padding:14px 32px;border-radius:8px;margin-bottom:32px;">Verify Email</a>
        <p style="font-size:13px;color:#6B7280;margin:0 0 8px 0;">Or copy and paste this URL into your browser:</p>
        <div style="background-color:#0B0E14;border:1px solid #30363D;border-radius:6px;padding:12px;word-break:break-all;">
          <a href="%s" style="font-family:'Courier New',monospace;font-size:12px;color:#2B8CEE;">%s</a>
        </div>
        <p style="font-size:12px;color:#6B7280;font-style:italic;margin:24px 0 0 0;">This link expires in 24 hours. If you did not sign up for Koolbase, you can safely ignore this email.</p>
      </td></tr>
    </table>
    <table border="0" cellpadding="0" cellspacing="0" width="520" style="margin-top:32px;">
      <tr><td align="center" style="font-size:12px;color:#6B7280;">
        <div style="margin-bottom:12px;">
          <a href="https://docs.koolbase.com" style="color:#6B7280;margin:0 8px;">Docs</a> •
          <a href="mailto:techfinityedge@gmail.com" style="color:#6B7280;margin:0 8px;">Support</a> •
          <a href="https://koolbase.com/terms" style="color:#6B7280;margin:0 8px;">Terms</a> •
          <a href="https://koolbase.com/privacy" style="color:#6B7280;margin:0 8px;">Privacy</a>
        </div>
        <p style="margin:0;">© 2025 Koolbase, Inc. All rights reserved.</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body></html>`, verifyURL, verifyURL, verifyURL)
}

func passwordResetEmailHTML(resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0a0c10; color: #f1f5f9; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; color: #2b8cee; }
</style>
</head>
<body style="background-color:#0a0c10;margin:0;padding:32px 16px;">
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
        <div style="background-color:rgba(43,140,238,0.1);border-radius:50%%;width:64px;height:64px;display:inline-block;text-align:center;line-height:64px;margin-bottom:24px;font-size:32px;">🔑</div>
        <h1 style="font-size:26px;font-weight:700;color:#ffffff;margin:0 0 12px 0;">Reset your password</h1>
        <p style="font-size:15px;color:#94a3b8;line-height:1.7;margin:0 0 32px 0;">We received a request to reset your password for your Koolbase account. Click the button below to choose a new one.</p>
        <a href="%s" style="display:inline-block;background-color:#2b8cee;color:#ffffff;font-size:16px;font-weight:600;padding:14px 36px;border-radius:8px;margin-bottom:32px;">Reset Password</a>
        <table border="0" cellpadding="0" cellspacing="0" width="100%%">
          <tr><td style="border-top:1px solid #1e2531;padding-top:28px;">
            <p style="font-size:13px;color:#64748b;text-align:center;margin:0 0 10px 0;">If the button doesn't work, copy and paste this link:</p>
            <div style="background-color:#0d1117;border:1px solid #1e2531;border-radius:6px;padding:12px;word-break:break-all;text-align:center;">
              <a href="%s" style="font-family:'Courier New',monospace;font-size:11px;color:#2b8cee;">%s</a>
            </div>
          </td></tr>
        </table>
        <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:24px;">
          <tr><td style="background-color:rgba(245,158,11,0.05);border:1px solid rgba(245,158,11,0.2);border-radius:8px;padding:14px 16px;">
            <p style="font-size:12px;color:#94a3b8;margin:0;line-height:1.6;"><strong style="color:#f59e0b;">Security note:</strong> This link will expire in 1 hour. If you didn't request a password reset, you can safely ignore this email.</p>
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
    <p style="margin:0;color:#334155;">© 2025 Koolbase Inc. All rights reserved.</p>
  </td></tr>
</table>
</body></html>`, resetURL, resetURL, resetURL)
}

func welcomeEmailHTML(name, dashboardURL, docsURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #101922; color: #f1f5f9; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; }
</style>
</head>
<body style="background-color:#101922;margin:0;padding:32px 16px;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center">
    <table border="0" cellpadding="0" cellspacing="0" width="600" style="background-color:#111827;border:1px solid #1f2937;border-radius:12px;overflow:hidden;">
      <!-- Logo -->
      <tr><td align="center" style="padding:40px 32px 24px 32px;">
        <table border="0" cellpadding="0" cellspacing="0">
          <tr>
            <td style="padding-right:10px;">
              <div style="width:32px;height:32px;background-color:#2b8cee;border-radius:6px;display:inline-block;text-align:center;line-height:32px;">
                <svg fill="none" height="20" width="20" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
                  <path d="M44 4H30.6666V17.3334H17.3334V30.6666H4V44H44V4Z" fill="white"/>
                </svg>
              </div>
            </td>
            <td style="font-size:22px;font-weight:700;color:#ffffff;">Koolbase</td>
          </tr>
        </table>
      </td></tr>
      <!-- Hero -->
      <tr><td align="center" style="padding:0 32px 24px 32px;">
        <h1 style="font-size:28px;font-weight:700;color:#ffffff;margin:0 0 16px 0;">Welcome to Koolbase, %s!</h1>
        <p style="font-size:16px;color:#94a3b8;line-height:1.7;margin:0;">We're excited to help you manage feature flags, remote config, and version enforcement with ease.</p>
      </td></tr>
      <!-- Steps -->
      <tr><td style="padding:0 32px 32px 32px;">
        <table border="0" cellpadding="0" cellspacing="0" width="100%%">
          <tr><td style="background-color:#1e293b;border:1px solid #1f2937;border-radius:8px;padding:16px;margin-bottom:12px;">
            <table border="0" cellpadding="0" cellspacing="0" width="100%%">
              <tr>
                <td width="44" style="padding-right:12px;">
                  <div style="width:40px;height:40px;background-color:rgba(43,140,238,0.1);border-radius:50%%;text-align:center;line-height:40px;font-size:18px;">⌨️</div>
                </td>
                <td>
                  <p style="font-size:14px;font-weight:600;color:#ffffff;margin:0 0 4px 0;">1. Install the SDK</p>
                  <p style="font-size:13px;color:#64748b;margin:0;">Integrate Koolbase into your Flutter app in minutes.</p>
                </td>
              </tr>
            </table>
          </td></tr>
          <tr><td style="height:8px;"></td></tr>
          <tr><td style="background-color:#1e293b;border:1px solid #1f2937;border-radius:8px;padding:16px;">
            <table border="0" cellpadding="0" cellspacing="0" width="100%%">
              <tr>
                <td width="44" style="padding-right:12px;">
                  <div style="width:40px;height:40px;background-color:rgba(43,140,238,0.1);border-radius:50%%;text-align:center;line-height:40px;font-size:18px;">🚩</div>
                </td>
                <td>
                  <p style="font-size:14px;font-weight:600;color:#ffffff;margin:0 0 4px 0;">2. Create your first flag</p>
                  <p style="font-size:13px;color:#64748b;margin:0;">Toggle features instantly without a redeploy.</p>
                </td>
              </tr>
            </table>
          </td></tr>
          <tr><td style="height:8px;"></td></tr>
          <tr><td style="background-color:#1e293b;border:1px solid #1f2937;border-radius:8px;padding:16px;">
            <table border="0" cellpadding="0" cellspacing="0" width="100%%">
              <tr>
                <td width="44" style="padding-right:12px;">
                  <div style="width:40px;height:40px;background-color:rgba(43,140,238,0.1);border-radius:50%%;text-align:center;line-height:40px;font-size:18px;">👥</div>
                </td>
                <td>
                  <p style="font-size:14px;font-weight:600;color:#ffffff;margin:0 0 4px 0;">3. Invite your team</p>
                  <p style="font-size:13px;color:#64748b;margin:0;">Collaborate with your engineers and product team.</p>
                </td>
              </tr>
            </table>
          </td></tr>
        </table>
      </td></tr>
      <!-- CTAs -->
      <tr><td style="padding:0 32px 40px 32px;">
        <a href="%s" style="display:block;width:100%%;padding:16px;background-color:#2b8cee;color:#ffffff;font-size:16px;font-weight:700;text-align:center;border-radius:8px;box-sizing:border-box;margin-bottom:12px;">Go to Dashboard</a>
        <a href="%s" style="display:block;width:100%%;padding:16px;background-color:#1e293b;color:#ffffff;font-size:15px;font-weight:600;text-align:center;border-radius:8px;box-sizing:border-box;">View Documentation</a>
      </td></tr>
      <!-- Footer -->
      <tr><td align="center" style="padding:32px;background-color:rgba(15,23,42,0.5);border-top:1px solid #1f2937;">
        <div style="margin-bottom:16px;">
          <a href="https://docs.koolbase.com" style="color:#64748b;font-size:12px;margin:0 10px;">Docs</a> •
          <a href="mailto:techfinityedge@gmail.com" style="color:#64748b;font-size:12px;margin:0 10px;">Support</a> •
          <a href="https://koolbase.com/terms" style="color:#64748b;font-size:12px;margin:0 10px;">Terms</a> •
          <a href="https://koolbase.com/privacy" style="color:#64748b;font-size:12px;margin:0 10px;">Privacy</a>
        </div>
        <p style="font-size:12px;color:#334155;margin:0;">© 2025 Koolbase Inc. All rights reserved.</p>
        <div style="margin-top:16px;display:inline-block;padding:4px 12px;background-color:#1e293b;border-radius:99px;font-size:10px;color:#64748b;font-weight:700;text-transform:uppercase;letter-spacing:0.05em;">Powered by Koolbase</div>
      </td></tr>
    </table>
  </td></tr>
</table>
</body></html>`, name, dashboardURL, docsURL)
}

func newLoginEmailHTML(email, ipAddress, location, device, loginTime string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #000000; color: #d1d5db; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; color: #3b82f6; }
</style>
</head>
<body style="background-color:#000000;margin:0;padding:48px 16px;">
<!-- Logo -->
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center" style="padding-bottom:32px;">
    <table border="0" cellpadding="0" cellspacing="0">
      <tr>
        <td style="padding-right:8px;">
          <div style="width:32px;height:32px;background-color:#2563eb;border-radius:6px;display:inline-block;text-align:center;line-height:32px;font-size:16px;color:white;font-weight:700;">K</div>
        </td>
        <td style="font-size:22px;font-weight:700;color:#ffffff;">Koolbase</td>
      </tr>
    </table>
  </td></tr>
</table>
<!-- Card -->
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center">
    <table border="0" cellpadding="0" cellspacing="0" width="560" style="background-color:#0f1115;border:1px solid #1f2937;border-radius:12px;overflow:hidden;">
      <tr><td align="center" style="padding:40px 40px 32px 40px;">
        <!-- Icon -->
        <div style="background-color:rgba(59,130,246,0.1);width:64px;height:64px;border-radius:50%%;display:inline-block;text-align:center;line-height:64px;font-size:28px;margin-bottom:24px;">🔐</div>
        <h2 style="font-size:28px;font-weight:700;color:#ffffff;margin:0 0 16px 0;">New login detected</h2>
        <p style="font-size:15px;color:#9ca3af;line-height:1.7;margin:0 0 32px 0;">We noticed a new login to your Koolbase account (<strong style="color:#e2e8f0;">%s</strong>). If this was you, you can safely ignore this email.</p>
        <!-- Details Table -->
        <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="background-color:#161b22;border:1px solid #1f2937;border-radius:8px;overflow:hidden;margin-bottom:32px;">
          <tr><td style="padding:16px 20px;border-bottom:1px solid #1f2937;">
            <table border="0" cellpadding="0" cellspacing="0" width="100%%">
              <tr>
                <td style="font-size:11px;color:#6b7280;text-transform:uppercase;letter-spacing:0.1em;font-weight:600;">IP Address</td>
                <td align="right" style="font-size:14px;color:#e2e8f0;">%s</td>
              </tr>
            </table>
          </td></tr>
          <tr><td style="padding:16px 20px;border-bottom:1px solid #1f2937;">
            <table border="0" cellpadding="0" cellspacing="0" width="100%%">
              <tr>
                <td style="font-size:11px;color:#6b7280;text-transform:uppercase;letter-spacing:0.1em;font-weight:600;">Location</td>
                <td align="right" style="font-size:14px;color:#e2e8f0;">%s</td>
              </tr>
            </table>
          </td></tr>
          <tr><td style="padding:16px 20px;border-bottom:1px solid #1f2937;">
            <table border="0" cellpadding="0" cellspacing="0" width="100%%">
              <tr>
                <td style="font-size:11px;color:#6b7280;text-transform:uppercase;letter-spacing:0.1em;font-weight:600;">Device</td>
                <td align="right" style="font-size:14px;color:#e2e8f0;">%s</td>
              </tr>
            </table>
          </td></tr>
          <tr><td style="padding:16px 20px;">
            <table border="0" cellpadding="0" cellspacing="0" width="100%%">
              <tr>
                <td style="font-size:11px;color:#6b7280;text-transform:uppercase;letter-spacing:0.1em;font-weight:600;">Time</td>
                <td align="right" style="font-size:14px;color:#e2e8f0;">%s</td>
              </tr>
            </table>
          </td></tr>
        </table>
        <!-- Security Note -->
        <table border="0" cellpadding="0" cellspacing="0" width="100%%">
          <tr><td style="background-color:rgba(245,158,11,0.05);border:1px solid rgba(245,158,11,0.2);border-radius:8px;padding:14px 16px;">
            <p style="font-size:12px;color:#fcd34d;margin:0;line-height:1.6;"><strong>Security note:</strong> If you don't recognize this activity, please reset your password immediately to secure your account.</p>
          </td></tr>
        </table>
      </td></tr>
    </table>
  </td></tr>
</table>
<!-- Footer -->
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:32px;">
  <tr><td align="center" style="font-size:12px;color:#4b5563;">
    <div style="margin-bottom:12px;">
      <a href="https://docs.koolbase.com" style="color:#4b5563;margin:0 10px;">Docs</a> •
      <a href="mailto:techfinityedge@gmail.com" style="color:#4b5563;margin:0 10px;">Support</a> •
      <a href="https://koolbase.com/terms" style="color:#4b5563;margin:0 10px;">Terms</a> •
      <a href="https://koolbase.com/privacy" style="color:#4b5563;margin:0 10px;">Privacy</a>
    </div>
    <p style="margin:0;">© 2025 Koolbase Inc. All rights reserved.</p>
  </td></tr>
</table>
</body></html>`, email, ipAddress, location, device, loginTime)
}

func usageLimitEmailHTML(currentUsage, limit string, upgradeURL, dashboardURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0b0f1a; color: #cbd5e1; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; }
</style>
</head>
<body style="background-color:#0b0f1a;margin:0;padding:48px 16px;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center">
    <table border="0" cellpadding="0" cellspacing="0" width="600" style="background-color:#111827;border:1px solid #1f2937;border-radius:12px;overflow:hidden;">
      <tr><td align="center" style="padding:40px 32px 24px 32px;">
        <table border="0" cellpadding="0" cellspacing="0" style="margin-bottom:24px;">
          <tr>
            <td style="padding-right:10px;">
              <div style="width:40px;height:40px;background-color:#2563eb;border-radius:8px;display:inline-block;text-align:center;line-height:40px;font-size:20px;color:white;font-weight:700;">K</div>
            </td>
            <td style="font-size:20px;font-weight:700;color:#ffffff;">Koolbase</td>
          </tr>
        </table>
        <h1 style="font-size:26px;font-weight:700;color:#ffffff;margin:0 0 12px 0;">Usage Limit Exceeded</h1>
        <p style="font-size:15px;color:#94a3b8;line-height:1.7;margin:0;">Your account has reached the maximum capacity for your current plan. Some services may be temporarily restricted until your plan is upgraded.</p>
      </td></tr>
      <!-- Usage Stats -->
      <tr><td style="padding:0 32px 32px 32px;">
        <div style="background-color:#1e293b;border:1px solid #334155;border-radius:8px;padding:24px;text-align:center;">
          <div style="display:inline-block;padding:4px 12px;background-color:rgba(239,68,68,0.1);border:1px solid rgba(239,68,68,0.2);border-radius:99px;font-size:11px;color:#f87171;font-weight:700;text-transform:uppercase;letter-spacing:0.05em;margin-bottom:16px;">Limit Reached</div>
          <p style="font-size:13px;color:#64748b;margin:0 0 8px 0;">Monthly Flag Evaluations</p>
          <p style="font-size:32px;font-weight:700;color:#ffffff;font-family:'Courier New',monospace;margin:0 0 16px 0;">%s</p>
          <div style="background-color:#334155;height:8px;border-radius:4px;overflow:hidden;margin-bottom:8px;">
            <div style="background-color:#ef4444;height:100%%;width:100%%;"></div>
          </div>
          <p style="font-size:12px;color:#64748b;font-family:'Courier New',monospace;margin:0;">Limit: %s</p>
        </div>
      </td></tr>
      <!-- CTAs -->
      <tr><td style="padding:0 32px 40px 32px;">
        <a href="%s" style="display:block;padding:16px;background-color:#3b82f6;color:#ffffff;font-size:16px;font-weight:600;text-align:center;border-radius:8px;margin-bottom:12px;">Upgrade Plan</a>
        <a href="%s" style="display:block;padding:16px;background-color:#1e293b;color:#cbd5e1;font-size:14px;font-weight:500;text-align:center;border-radius:8px;">View Usage Dashboard</a>
      </td></tr>
      <tr><td align="center" style="padding:24px 32px;background-color:rgba(15,23,42,0.5);border-top:1px solid #1f2937;">
        <div style="margin-bottom:12px;">
          <a href="https://docs.koolbase.com" style="color:#475569;font-size:12px;margin:0 8px;">Docs</a> •
          <a href="mailto:techfinityedge@gmail.com" style="color:#475569;font-size:12px;margin:0 8px;">Support</a> •
          <a href="https://koolbase.com/terms" style="color:#475569;font-size:12px;margin:0 8px;">Terms</a> •
          <a href="https://koolbase.com/privacy" style="color:#475569;font-size:12px;margin:0 8px;">Privacy</a>
        </div>
        <p style="font-size:12px;color:#334155;margin:0;">© 2025 Koolbase Inc. All rights reserved.</p>
        <div style="margin-top:12px;display:inline-block;padding:4px 12px;background-color:#1e293b;border-radius:99px;font-size:10px;color:#475569;font-weight:700;text-transform:uppercase;">Powered by Koolbase</div>
      </td></tr>
    </table>
  </td></tr>
</table>
</body></html>`, currentUsage, limit, upgradeURL, dashboardURL)
}

func paymentFailedEmailHTML(last4, updateURL, invoiceURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0a0a0a; color: #d1d5db; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; }
</style>
</head>
<body style="background-color:#0a0a0a;margin:0;padding:48px 16px;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center" style="padding-bottom:24px;">
    <table border="0" cellpadding="0" cellspacing="0">
      <tr>
        <td style="padding-right:8px;">
          <svg fill="none" height="32" width="32" viewBox="0 0 32 32" xmlns="http://www.w3.org/2000/svg">
            <path d="M16 4L4 10V22L16 28L28 22V10L16 4Z" stroke="#2b8cee" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M16 12L10 15V20L16 23L22 20V15L16 12Z" fill="#2b8cee"/>
          </svg>
        </td>
        <td style="font-size:20px;font-weight:700;color:#ffffff;">Koolbase</td>
      </tr>
    </table>
  </td></tr>
</table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center">
    <table border="0" cellpadding="0" cellspacing="0" width="560" style="background-color:#171717;border:1px solid #262626;border-radius:8px;overflow:hidden;">
      <tr><td align="center" style="padding:40px 40px 32px 40px;">
        <div style="background-color:rgba(239,68,68,0.1);width:56px;height:56px;border-radius:50%%;display:inline-block;text-align:center;line-height:56px;font-size:28px;margin-bottom:24px;">⚠️</div>
        <h1 style="font-size:24px;font-weight:600;color:#ffffff;margin:0 0 24px 0;">Action Required: Payment Failed</h1>
        <p style="font-size:15px;color:#9ca3af;line-height:1.7;margin:0 0 16px 0;text-align:left;">Hi there,</p>
        <p style="font-size:15px;color:#9ca3af;line-height:1.7;margin:0 0 24px 0;text-align:left;">We're reaching out to let you know that our recent attempt to charge your card ending in <strong style="color:#ffffff;font-family:'Courier New',monospace;">•••• %s</strong> for your Koolbase subscription was unsuccessful.</p>
        <!-- Warning -->
        <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-bottom:24px;">
          <tr><td style="background-color:rgba(245,158,11,0.05);border:1px solid rgba(245,158,11,0.2);border-radius:8px;padding:14px 16px;">
            <p style="font-size:13px;color:#fbbf24;margin:0;line-height:1.6;"><strong>⚠️</strong> Your services may be interrupted if the payment is not resolved within the next <strong>48 hours</strong>.</p>
          </td></tr>
        </table>
        <p style="font-size:15px;color:#9ca3af;line-height:1.7;margin:0 0 32px 0;text-align:left;">To keep your projects running smoothly, please update your payment information.</p>
        <a href="%s" style="display:block;padding:14px 32px;background-color:#2b8cee;color:#ffffff;font-size:16px;font-weight:600;text-align:center;border-radius:8px;margin-bottom:12px;">Update Billing Info</a>
        <a href="%s" style="display:block;padding:14px 32px;border:1px solid #262626;color:#9ca3af;font-size:14px;font-weight:500;text-align:center;border-radius:8px;">View Invoice</a>
      </td></tr>
    </table>
  </td></tr>
</table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:32px;">
  <tr><td align="center" style="font-size:12px;color:#4b5563;">
    <div style="margin-bottom:12px;">
      <a href="https://docs.koolbase.com" style="color:#4b5563;margin:0 10px;">Docs</a> •
      <a href="mailto:techfinityedge@gmail.com" style="color:#4b5563;margin:0 10px;">Support</a> •
      <a href="https://koolbase.com/terms" style="color:#4b5563;margin:0 10px;">Terms</a> •
      <a href="https://koolbase.com/privacy" style="color:#4b5563;margin:0 10px;">Privacy</a>
    </div>
    <p style="margin:0;">© 2025 Koolbase Inc. All rights reserved.</p>
  </td></tr>
</table>
</body></html>`, last4, updateURL, invoiceURL)
}

func subscriptionCancelledEmailHTML(accessUntil, reactivateURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #0a0c10; color: #d1d5db; margin: 0; padding: 0; }
  table { border-collapse: collapse; }
  a { text-decoration: none; }
</style>
</head>
<body style="background-color:#0a0c10;margin:0;padding:48px 16px;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center" style="padding-bottom:24px;">
    <table border="0" cellpadding="0" cellspacing="0">
      <tr>
        <td style="padding-right:8px;">
          <div style="width:32px;height:32px;background-color:#3b82f6;border-radius:6px;display:inline-block;text-align:center;line-height:32px;color:white;font-size:16px;font-weight:700;">K</div>
        </td>
        <td style="font-size:20px;font-weight:700;color:#ffffff;">Koolbase</td>
      </tr>
    </table>
  </td></tr>
</table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%">
  <tr><td align="center">
    <table border="0" cellpadding="0" cellspacing="0" width="560" style="background-color:#11141a;border:1px solid #1f2937;border-radius:12px;overflow:hidden;">
      <tr><td align="center" style="padding:40px 40px 32px 40px;">
        <div style="width:48px;height:48px;background-color:#1f2937;border-radius:50%%;display:inline-block;text-align:center;line-height:48px;font-size:24px;margin-bottom:24px;">😢</div>
        <h1 style="font-size:26px;font-weight:700;color:#ffffff;margin:0 0 24px 0;">Subscription Cancelled</h1>
        <p style="font-size:15px;color:#9ca3af;line-height:1.7;margin:0 0 16px 0;text-align:left;">Hi there,</p>
        <p style="font-size:15px;color:#9ca3af;line-height:1.7;margin:0 0 24px 0;text-align:left;">We're sorry to see you go! This email confirms that your Koolbase subscription has been cancelled. You will continue to have full access to your projects and features until the end of your current billing cycle on <strong style="color:#ffffff;">%s</strong>.</p>
        <!-- Info Box -->
        <table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-bottom:24px;">
          <tr><td style="background-color:#1f2937;border:1px solid #374151;border-radius:8px;padding:14px 16px;">
            <p style="font-size:13px;color:#93c5fd;margin:0;line-height:1.6;"><strong>Note:</strong> Your project data and configurations will be preserved for 30 days. After this period, your data will be permanently deleted unless the subscription is reactivated.</p>
          </td></tr>
        </table>
        <p style="font-size:15px;color:#9ca3af;line-height:1.7;margin:0 0 32px 0;text-align:left;">If you changed your mind or cancelled by mistake, you can reactivate your subscription at any time.</p>
        <a href="%s" style="display:block;padding:14px 32px;background-color:#3b82f6;color:#ffffff;font-size:16px;font-weight:600;text-align:center;border-radius:8px;margin-bottom:16px;">Reactivate Subscription</a>
        <a href="mailto:techfinityedge@gmail.com" style="display:block;font-size:13px;color:#6b7280;text-align:center;text-decoration:underline;">Tell us why you left</a>
      </td></tr>
    </table>
  </td></tr>
</table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:32px;">
  <tr><td align="center" style="font-size:12px;color:#4b5563;">
    <div style="margin-bottom:12px;">
      <a href="https://docs.koolbase.com" style="color:#4b5563;margin:0 10px;text-transform:uppercase;letter-spacing:0.05em;">Docs</a> •
      <a href="mailto:techfinityedge@gmail.com" style="color:#4b5563;margin:0 10px;text-transform:uppercase;letter-spacing:0.05em;">Support</a> •
      <a href="https://koolbase.com/terms" style="color:#4b5563;margin:0 10px;text-transform:uppercase;letter-spacing:0.05em;">Terms</a> •
      <a href="https://koolbase.com/privacy" style="color:#4b5563;margin:0 10px;text-transform:uppercase;letter-spacing:0.05em;">Privacy</a>
    </div>
    <p style="margin:0;">© 2025 Koolbase Inc. All rights reserved.</p>
    <div style="margin-top:12px;display:inline-block;padding:4px 12px;background-color:#1e293b;border-radius:99px;font-size:10px;color:#475569;font-weight:700;text-transform:uppercase;border:1px solid #1f2937;">Powered by Koolbase</div>
  </td></tr>
</table>
</body></html>`, accessUntil, reactivateURL)
}
