package mailer

import (
	"log"

	"github.com/troubleman96/ziadapos/internal/config"
	"gopkg.in/gomail.v2"
)

// Mailer sends transactional emails over Zoho SMTP. All app emails (welcome,
// verification, daily reports, receipts) go through this single path so
// behaviour is consistent in dev and production.
type Mailer struct {
	from     string
	fromName string
	smtp     *gomail.Dialer
	enabled  bool
	mode     string
}

// New builds a Mailer from config. When no SMTP backend is configured the
// mailer runs in "log-only" mode (emails are logged, not sent) so local
// development never crashes on missing credentials.
func New(cfg *config.Config) *Mailer {
	m := &Mailer{
		from:     cfg.SMTPFrom,
		fromName: cfg.SMTPFromName,
	}

	switch {
	case cfg.SMTPUser != "" && cfg.SMTPPassword != "":
		m.mode = "smtp"
		m.smtp = gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword)
		m.enabled = true
	default:
		m.mode = "log"
		log.Println("mailer: no email backend configured - running in log-only mode")
	}
	return m
}

// Send delivers an email to a recipient.
func (m *Mailer) Send(to, subject, html string) error {
	if !m.enabled {
		log.Printf("mailer [dry-run] to=%s subject=%q", to, subject)
		return nil
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.smtp.Username)
	msg.SetAddressHeader("From", m.from, m.fromName)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", html)
	if err := m.smtp.DialAndSend(msg); err != nil {
		return err
	}
	log.Printf("mailer: sent to=%s subject=%q", to, subject)
	return nil
}

// Enabled reports whether real email delivery is configured.
func (m *Mailer) Enabled() bool { return m.enabled }

// Layout wraps a body HTML fragment in the shared branded template.
func (m *Mailer) Layout(title, body string) string {
	return "ZiadaPOS " + title + "\n\n" + body
}
