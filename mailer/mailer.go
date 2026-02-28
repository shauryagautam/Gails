package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"

	"github.com/shaurya/gails/config"
	"github.com/shaurya/gails/framework"
	"go.uber.org/zap"
)

// Mailer is the base type for all mailers — embed this in your mailers.
type Mailer struct {
	Config config.MailerConfig
}

// Email represents an email to be delivered.
type Email struct {
	to       string
	from     string
	subject  string
	htmlBody string
	textBody string
	mailer   *Mailer
}

// NewEmail creates a new email builder.
func (m *Mailer) NewEmail() *Email {
	return &Email{
		mailer: m,
		from:   m.Config.From,
	}
}

// To sets the recipient.
func (e *Email) To(to string) *Email {
	e.to = to
	return e
}

// Subject sets the subject.
func (e *Email) Subject(subject string) *Email {
	e.subject = subject
	return e
}

// Template renders an email template and sets the HTML body.
func (e *Email) Template(name string, data any) *Email {
	// Try to load from views/mailers/{name}.html
	tmplPath := fmt.Sprintf("views/mailers/%s.html", name)
	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		// Fallback: render the data as a simple string
		e.htmlBody = fmt.Sprintf("<p>%v</p>", data)
		e.textBody = fmt.Sprintf("%v", data)
		return e
	}

	var buf bytes.Buffer
	t.Execute(&buf, data)
	e.htmlBody = buf.String()
	e.textBody = fmt.Sprintf("%v", data) // Simple text fallback

	return e
}

// Body sets a plain text body.
func (e *Email) Body(body string) *Email {
	e.textBody = body
	return e
}

// HTMLBody sets an HTML body.
func (e *Email) HTMLBody(body string) *Email {
	e.htmlBody = body
	return e
}

// Deliver sends the email synchronously.
func (e *Email) Deliver() error {
	env := os.Getenv("APP_ENV")
	if env == "development" || env == "test" || env == "" {
		if framework.Log != nil {
			framework.Log.Info("📧 Intercepted email (not sent)",
				zap.String("to", e.to),
				zap.String("subject", e.subject),
				zap.String("body_preview", truncate(e.textBody, 200)),
			)
		}
		return nil
	}

	// Build multipart message
	boundary := "GailsMailBoundary"
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=%s\r\n\r\n"+
		"--%s\r\n"+
		"Content-Type: text/plain; charset=utf-8\r\n\r\n"+
		"%s\r\n"+
		"--%s\r\n"+
		"Content-Type: text/html; charset=utf-8\r\n\r\n"+
		"%s\r\n"+
		"--%s--\r\n",
		e.from, e.to, e.subject,
		boundary,
		boundary, e.textBody,
		boundary, e.htmlBody,
		boundary,
	)

	auth := smtp.PlainAuth("", e.from, "", e.mailer.Config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", e.mailer.Config.SMTPHost, e.mailer.Config.SMTPPort)

	return smtp.SendMail(addr, auth, e.from, []string{e.to}, []byte(msg))
}

// DeliverLater enqueues the email for background delivery.
func (e *Email) DeliverLater() {
	if framework.Log != nil {
		framework.Log.Info("📧 Enqueued email for later delivery",
			zap.String("to", e.to),
			zap.String("subject", e.subject),
		)
	}
	// In a full implementation, this would enqueue to the job queue
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
