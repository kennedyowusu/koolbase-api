package billing

import "fmt"

func alertEmailHTML(orgName, resource string, current, limit, pct int, appURL string) string {
	resourceLabel := map[string]string{
		"environments": "Environments",
		"flags":        "Feature Flags",
		"configs":      "Remote Config values",
		"members":      "Team Members",
		"functions":    "Functions",
		"secrets":      "Secrets",
	}
	label := resourceLabel[resource]
	if label == "" {
		label = resource
	}

	barWidth := pct
	if barWidth > 100 {
		barWidth = 100
	}
	barColor := "#f59e0b"
	if pct >= 90 {
		barColor = "#ef4444"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"/><meta name="viewport" content="width=device-width,initial-scale=1.0"/></head>
<body style="margin:0;padding:0;background:#0a0a0b;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#0a0a0b;padding:40px 20px;">
<tr><td align="center">
<table width="100%%" cellpadding="0" cellspacing="0" style="max-width:560px;">

  <tr><td style="padding-bottom:28px;">
    <span style="font-size:20px;font-weight:900;color:#ffffff;letter-spacing:-0.5px;">Koolbase</span>
  </td></tr>

  <tr><td style="background:#101922;border:1px solid #1e293b;border-radius:16px;padding:32px;">

    <p style="margin:0 0 6px;font-size:18px;font-weight:800;color:#ffffff;">Usage limit approaching</p>
    <p style="margin:0 0 24px;font-size:14px;color:#64748b;">%s</p>

    <p style="margin:0 0 20px;font-size:15px;color:#94a3b8;line-height:1.6;">
      Your organization <strong style="color:#ffffff;">%s</strong> has used
      <strong style="color:#ffffff;">%d of %d %s</strong> — that's <strong style="color:%s;">%d%%</strong> of your plan limit.
    </p>

    <div style="background:#1e293b;border-radius:8px;height:8px;overflow:hidden;margin-bottom:6px;">
      <div style="background:%s;height:8px;width:%d%%;border-radius:8px;"></div>
    </div>
    <p style="margin:0 0 24px;font-size:12px;color:#475569;text-align:right;">%d / %d used</p>

    <div style="background:#0f172a;border:1px solid #1e293b;border-radius:12px;padding:16px;margin-bottom:24px;">
      <p style="margin:0 0 6px;font-size:11px;font-weight:700;color:#475569;text-transform:uppercase;letter-spacing:0.05em;">What you can do</p>
      <p style="margin:0;font-size:14px;color:#94a3b8;line-height:1.6;">
        Upgrade your plan to increase your limits, or remove unused %s to free up capacity.
      </p>
    </div>

    <a href="%s/settings"
       style="display:inline-block;background:#2b8cee;color:#ffffff;font-size:14px;font-weight:700;padding:12px 24px;border-radius:8px;text-decoration:none;">
      View Usage &amp; Upgrade →
    </a>

  </td></tr>

  <tr><td style="padding-top:20px;">
    <p style="margin:0;font-size:12px;color:#334155;line-height:1.6;">
      You're receiving this as the owner of a Koolbase organization.<br/>
      <a href="https://docs.koolbase.com" style="color:#475569;">docs.koolbase.com</a>
    </p>
  </td></tr>

</table>
</td></tr>
</table>
</body>
</html>`,
		label,
		orgName, current, limit, label, barColor, pct,
		barColor, barWidth,
		current, limit,
		label,
		appURL,
	)
}
