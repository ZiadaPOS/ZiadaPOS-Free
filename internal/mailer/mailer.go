package mailer

import (
	"log"

	"github.com/troubleman96/ziadapos/internal/config"
	"gopkg.in/gomail.v2"
)

// Mailer sends transactional emails over SMTP. All app emails (welcome,
// verification, daily reports, receipts) go through this single path so
// behaviour is consistent in dev and production.
type Mailer struct {
	from    string
	fromName string
	dialer  *gomail.Dialer
	enabled bool
}

// New builds a Mailer from config. When SMTP credentials are absent the
// mailer runs in "log-only" mode (emails are logged, not sent) so local
// development never crashes on missing credentials.
func New(cfg *config.Config) *Mailer {
	enabled := cfg.SMTPUser != "" && cfg.SMTPPassword != ""
	if !enabled {
		log.Println("mailer: SMTP not configured - running in log-only mode")
	}
	return &Mailer{
		from:     cfg.SMTPFrom,
		fromName: cfg.SMTPFromName,
		dialer:   gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword),
		enabled:  enabled,
	}
}

// Send delivers an email to a recipient.
func (m *Mailer) Send(to, subject, html string) error {
	if !m.enabled {
		log.Printf("mailer [dry-run] to=%s subject=%q", to, subject)
		return nil
	}
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.dialer.Username)
	msg.SetAddressHeader("From", m.from, m.fromName)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", html)
	if err := m.dialer.DialAndSend(msg); err != nil {
		return err
	}
	log.Printf("mailer: sent to=%s subject=%q", to, subject)
	return nil
}

// Enabled reports whether real SMTP delivery is configured.
func (m *Mailer) Enabled() bool { return m.enabled }

// Layout wraps body HTML in the shared branded template.
func (m *Mailer) Layout(title, body string) string {
	return `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>` + title + `</title></head>
<body style="margin:0;padding:0;background:#f3f4f6;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f3f4f6;padding:24px 0;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;overflow:hidden;border:1px solid #e5e7eb;">
        <tr>
          <td style="background:#4f46e5;padding:20px 28px;">
            <span style="color:#ffffff;font-size:20px;font-weight:bold;">Ziada<span style="color:#c7d2fe;">POS</span></span>
          </td>
        </tr>
        <tr><td style="padding:28px;color:#111827;font-size:15px;line-height:1.6;">
          ` + body + `
        </td></tr>
        <tr><td style="padding:20px 28px;background:#f9fafb;border-top:1px solid #e5e7eb;color:#6b7280;font-size:12px;">
          ZiadaPOS &middot; POS for Tanzanian shops &middot; Powered by CamelTech
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`
}
