package mailer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mailersend/mailersend-go"
)

type MailerSendClient struct {
	client  *mailersend.Mailersend
	from    mailersend.From
	enabled bool
}

func NewMailerSend(apiKey, fromName, fromEmail string) *MailerSendClient {
	m := &MailerSendClient{
		enabled: apiKey != "" && fromEmail != "",
		from: mailersend.From{
			Name:  fromName,
			Email: fromEmail,
		},
	}
	
	if m.enabled {
		m.client = mailersend.NewMailersend(apiKey)
	}
	
	return m
}

func (m *MailerSendClient) SendVerificationEmail(toEmail, toName, verifyURL, token string) error {
	if !m.enabled {
		return fmt.Errorf("MailerSend not configured")
	}
	
	subject := "Verify your LuxSUV account"
	html := fmt.Sprintf(`
		<h2>Welcome to LuxSUV!</h2>
		<p>Hi %s,</p>
		<p>Please verify your email address by clicking the link below:</p>
		<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
		<p>Or use this verification code: <strong>%s</strong></p>
		<p>This link will expire in 2 hours.</p>
		<p>If you didn't create an account with us, please ignore this email.</p>
	`, toName, verifyURL, token)
	
	text := fmt.Sprintf("Please verify your email by clicking this link: %s\n\nOr use this verification code: %s", verifyURL, token)
	
	return m.sendEmail(toEmail, toName, subject, text, html)
}

func (m *MailerSendClient) SendGuestAccessEmail(email, code, magicLink string) error {
	if !m.enabled {
		return fmt.Errorf("MailerSend not configured")
	}
	
	subject := "Your LuxSUV access code"
	html := fmt.Sprintf(`
		<h2>Your LuxSUV Access Code</h2>
		<p>Your verification code is: <strong style="font-size: 24px; color: #4CAF50;">%s</strong></p>
		<p>Or click the link below to access directly:</p>
		<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Access Account</a></p>
		<p>This code will expire in 15 minutes.</p>
	`, code, magicLink)
	
	text := fmt.Sprintf("Your access code is: %s\n\nOr click this link to access directly: %s", code, magicLink)
	
	return m.sendEmail(email, "", subject, text, html)
}

func (m *MailerSendClient) sendEmail(toEmail, toName, subject, text, html string) error {
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
	
	_, err := m.client.Email.Send(ctx, msg)
	return err
}