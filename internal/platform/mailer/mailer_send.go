// internal/platform/mailer/mailer.go
package mailer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mailersend/mailersend-go"
)

type Mailer struct {
	client  *mailersend.Mailersend
	from    mailersend.From
	Enabled bool
}

func NewMailer(apiKey, fromName, fromEmail string) *Mailer {
	m := &Mailer{
		Enabled: apiKey != "" && fromEmail != "",
		from: mailersend.From{
			Name:  fromName,
			Email: fromEmail,
		},
	}
	if m.Enabled {
		m.client = mailersend.NewMailersend(apiKey)
	}
	return m
}

func (m *Mailer) Send(toEmail, toName, subject, text, html string) (string, error) {
	if !m.Enabled {
		return "", errors.New("mailer disabled (missing MAILERSEND_API_KEY or MAILER_FROM)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg := m.client.Email.NewMessage()
	msg.SetFrom(m.from)
	msg.SetRecipients([]mailersend.Recipient{{Name: toName, Email: toEmail}})
	msg.SetSubject(subject)
	if strings.TrimSpace(text) != "" {
		msg.SetText(text)
	}
	if strings.TrimSpace(html) != "" {
		msg.SetHTML(html)
	}

	res, err := m.client.Email.Send(ctx, msg)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("mailersend error: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	// MailerSend uses X-Message-Id
	return res.Header.Get("X-Message-Id"), nil
}

func (m *Mailer) SendGuestAccess(email, code, link string) error {
	subject := "Your LuxSuv booking access link"
	text := fmt.Sprintf("Your code is %s. Or click the link: %s", code, link)
	html := fmt.Sprintf(`<p>Your code is <b>%s</b></p><p>Or click <a href="%s">%s</a></p>`, code, link, link)
	_, err := m.Send(email, "", subject, text, html)
	return err
}
