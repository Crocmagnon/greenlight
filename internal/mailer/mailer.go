// Package mailer provides tools to send emails.
package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

//go:embed "templates"
var templateFS embed.FS

// A Mailer is capable of sending emails.
// The zero value is not usable. Use New.
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

// New creates a new Mailer.
func New(host string, port int, username, password, sender string) Mailer { //nolint:revive
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second //nolint:revive,gomnd // not used anywhere else

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// Send sends an email to the recipient, using the template described in templateFile.
// The data is passed to the template.
// Template files are expected to define "subject", "plainBody", and "htmlBody".
func (m Mailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return fmt.Errorf("parsing filesystem: %w", err)
	}

	subject := new(bytes.Buffer)
	plainBody := new(bytes.Buffer)
	htmlBody := new(bytes.Buffer)

	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return fmt.Errorf("templating subject: %w", err)
	}

	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return fmt.Errorf("templating plainBody: %w", err)
	}

	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return fmt.Errorf("templating htmlBody: %w", err)
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	const maxRetry = 3
	for i := 1; i <= maxRetry; i++ {
		err = m.dialer.DialAndSend(msg)
		if nil == err {
			// if NO error
			return nil
		}

		time.Sleep(500 * time.Millisecond) //nolint:gomnd
	}

	return fmt.Errorf("sending msg: %w", err)
}
