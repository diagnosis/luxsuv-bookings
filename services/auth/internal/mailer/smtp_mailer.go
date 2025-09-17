package mailer

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPMailer struct {
	Host   string
	Port   int
	From   string
	User   string
	Pass   string
	UseTLS bool
}

func NewSMTPMailer(host string, port int, from, user, pass string, useTLS bool) *SMTPMailer {
	return &SMTPMailer{
		Host:   strings.TrimSpace(host),
		Port:   port,
		From:   strings.TrimSpace(from),
		User:   strings.TrimSpace(user),
		Pass:   strings.TrimSpace(pass),
		UseTLS: useTLS,
	}
}

func (s *SMTPMailer) SendVerificationEmail(toEmail, toName, verifyURL, token string) error {
	subject := "Verify your LuxSUV account"
	text := fmt.Sprintf("Please verify your email by clicking this link: %s\n\nOr use this verification code: %s", verifyURL, token)
	html := fmt.Sprintf(`
		<h2>Welcome to LuxSUV!</h2>
		<p>Hi %s,</p>
		<p>Please verify your email address by clicking the link below:</p>
		<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
		<p>Or use this verification code: <strong>%s</strong></p>
		<p>This link will expire in 2 hours.</p>
		<p>If you didn't create an account with us, please ignore this email.</p>
	`, toName, verifyURL, token)
	
	return s.sendEmail(toEmail, toName, subject, text, html)
}

func (s *SMTPMailer) SendGuestAccessEmail(email, code, magicLink string) error {
	subject := "Your LuxSUV access code"
	text := fmt.Sprintf("Your access code is: %s\n\nOr click this link to access directly: %s", code, magicLink)
	html := fmt.Sprintf(`
		<h2>Your LuxSUV Access Code</h2>
		<p>Your verification code is: <strong style="font-size: 24px; color: #4CAF50;">%s</strong></p>
		<p>Or click the link below to access directly:</p>
		<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Access Account</a></p>
		<p>This code will expire in 15 minutes.</p>
	`, code, magicLink)
	
	return s.sendEmail(email, "", subject, text, html)
}

func (s *SMTPMailer) sendEmail(toEmail, toName, subject, text, html string) error {
	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		return fmt.Errorf("empty recipient email")
	}
	
	var buf bytes.Buffer
	boundary := "mixed-boundary"
	
	fmt.Fprintf(&buf, "From: %s\r\n", s.From)
	fmt.Fprintf(&buf, "To: %s\r\n", toEmail)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)
	
	// Text part
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&buf, "%s\r\n\r\n", text)
	
	// HTML part
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/html; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&buf, "%s\r\n\r\n", html)
	
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	
	// Mailpit or development SMTP (no auth, no TLS)
	if !s.UseTLS && s.User == "" {
		return smtp.SendMail(addr, nil, s.From, []string{toEmail}, buf.Bytes())
	}
	
	// Production SMTP with authentication
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Pass, s.Host)
	}
	
	// Try plain SMTP first (with STARTTLS if supported)
	if err := smtp.SendMail(addr, auth, s.From, []string{toEmail}, buf.Bytes()); err == nil {
		return nil
	}
	
	// Fallback to implicit TLS (port 465)
	if s.UseTLS {
		tlsCfg := &tls.Config{ServerName: s.Host, InsecureSkipVerify: false}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		defer conn.Close()
		
		c, err := smtp.NewClient(conn, s.Host)
		if err != nil {
			return err
		}
		defer c.Quit()
		
		if s.User != "" {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		
		if err := c.Mail(s.From); err != nil {
			return err
		}
		if err := c.Rcpt(toEmail); err != nil {
			return err
		}
		
		w, err := c.Data()
		if err != nil {
			return err
		}
		
		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}
		
		return w.Close()
	}
	
	return fmt.Errorf("smtp send failed")
}