package projectauth

import (
	"fmt"
	"time"
)

func currentYear() string {
	return fmt.Sprintf("%d", time.Now().Year())
}

func verifyEmailHTML(verifyURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"/><meta content="width=device-width,initial-scale=1.0" name="viewport"/>
<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#0a0c10;color:#f1f5f9;margin:0;padding:0;}</style>
</head>
<body style="background:#0a0c10;padding:48px 16px;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%"><tr><td align="center" style="padding-bottom:24px;">
<table border="0" cellpadding="0" cellspacing="0"><tr>
<td style="padding-right:10px;"><svg fill="none" height="32" width="32" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg"><path d="M44 4H30.6666V17.3334H17.3334V30.6666H4V44H44V4Z" fill="#2b8cee"/></svg></td>
<td style="font-size:22px;font-weight:700;color:#ffffff;">Koolbase</td>
</tr></table></td></tr></table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%"><tr><td align="center">
<table border="0" cellpadding="0" cellspacing="0" width="560" style="background:#111418;border:1px solid #1e2531;border-radius:12px;">
<tr><td style="padding:40px;">
<h1 style="font-size:24px;font-weight:700;color:#fff;margin:0 0 12px 0;">Verify your email</h1>
<p style="font-size:15px;color:#94a3b8;line-height:1.7;margin:0 0 28px 0;">Click the button below to verify your email address and activate your account.</p>
<a href="%s" style="display:inline-block;background:#2b8cee;color:#fff;font-size:16px;font-weight:600;padding:14px 36px;border-radius:8px;text-decoration:none;">Verify Email</a>
<p style="font-size:12px;color:#64748b;margin:24px 0 0 0;">This link expires in 24 hours. If you didn't create an account, you can ignore this email.</p>
</td></tr></table></td></tr></table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:32px;"><tr><td align="center" style="font-size:12px;color:#475569;">
<p style="margin:0;">© %s Koolbase Inc. All rights reserved.</p>
</td></tr></table>
</body></html>`, verifyURL, currentYear())
}

func resetEmailHTML(resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"/><meta content="width=device-width,initial-scale=1.0" name="viewport"/>
<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#0a0c10;color:#f1f5f9;margin:0;padding:0;}</style>
</head>
<body style="background:#0a0c10;padding:48px 16px;">
<table border="0" cellpadding="0" cellspacing="0" width="100%%"><tr><td align="center" style="padding-bottom:24px;">
<table border="0" cellpadding="0" cellspacing="0"><tr>
<td style="padding-right:10px;"><svg fill="none" height="32" width="32" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg"><path d="M44 4H30.6666V17.3334H17.3334V30.6666H4V44H44V4Z" fill="#2b8cee"/></svg></td>
<td style="font-size:22px;font-weight:700;color:#ffffff;">Koolbase</td>
</tr></table></td></tr></table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%"><tr><td align="center">
<table border="0" cellpadding="0" cellspacing="0" width="560" style="background:#111418;border:1px solid #1e2531;border-radius:12px;">
<tr><td style="padding:40px;">
<h1 style="font-size:24px;font-weight:700;color:#fff;margin:0 0 12px 0;">Reset your password</h1>
<p style="font-size:15px;color:#94a3b8;line-height:1.7;margin:0 0 28px 0;">Click the button below to reset your password. This link expires in 1 hour.</p>
<a href="%s" style="display:inline-block;background:#2b8cee;color:#fff;font-size:16px;font-weight:600;padding:14px 36px;border-radius:8px;text-decoration:none;">Reset Password</a>
<p style="font-size:12px;color:#64748b;margin:24px 0 0 0;">If you didn't request a password reset, you can ignore this email.</p>
</td></tr></table></td></tr></table>
<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="margin-top:32px;"><tr><td align="center" style="font-size:12px;color:#475569;">
<p style="margin:0;">© %s Koolbase Inc. All rights reserved.</p>
</td></tr></table>
</body></html>`, resetURL, currentYear())
}
