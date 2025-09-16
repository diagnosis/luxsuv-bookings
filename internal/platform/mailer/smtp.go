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
	UseTLS bool // false for Mailpit on 1025
}

func NewSMTPMailer(host string, port int, from string, user string, pass string, useTLS bool) *SMTPMailer {
	return &SMTPMailer{
		Host:   strings.TrimSpace(host),
		Port:   port,
		From:   strings.TrimSpace(from),
		User:   strings.TrimSpace(user),
		Pass:   strings.TrimSpace(pass),
		UseTLS: useTLS,
	}
}

func (s *SMTPMailer) Send(toEmail, toName, subject, text, html string) (string, error) {
	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		return "", fmt.Errorf("empity recipient email")
	}

	var buf bytes.Buffer
	boundary := "mixed-boundary"
	fmt.Fprintf(&buf, "From: %s\r\n", s.From)
	fmt.Fprintf(&buf, "To: %s\r\n", toEmail)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)

	// text part
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&buf, "%s\r\n\r\n", text)

	// html part
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/html; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&buf, "%s\r\n\r\n", html)

	fmt.Fprintf(&buf, "--%s--\r\n", boundary)

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	// Mailpit on 1025: no auth, no TLS
	if !s.UseTLS && s.User == "" {
		return "", smtp.SendMail(addr, nil, s.From, []string{toEmail}, buf.Bytes())
	}

	// STARTTLS / AUTH path (not needed for Mailpit, but handy for staging SMTP)
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Pass, s.Host)
	}

	// Try plain SendMail first (it will STARTTLS if advertised)
	if err := smtp.SendMail(addr, auth, s.From, []string{toEmail}, buf.Bytes()); err == nil {
		return "", nil
	}

	// Fallback to implicit TLS (e.g., port 465) if requested
	if s.UseTLS {
		tlsCfg := &tls.Config{ServerName: s.Host, InsecureSkipVerify: true}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return "", err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, s.Host)
		if err != nil {
			return "", err
		}
		defer c.Quit()

		if s.User != "" {
			if err := c.Auth(auth); err != nil {
				return "", err
			}
		}
		if err := c.Mail(s.From); err != nil {
			return "", err
		}
		if err := c.Rcpt(toEmail); err != nil {
			return "", err
		}
		w, err := c.Data()
		if err != nil {
			return "", err
		}
		if _, err := w.Write(buf.Bytes()); err != nil {
			return "", err
		}
		return "", w.Close()
	}

	return "", fmt.Errorf("smtp send failed")
}

func (s *SMTPMailer) SendGuestAccess(email, code, link string) error {
	subject := "Your LuxSuv guest access code"
	text := fmt.Sprintf("Your access code is %s\nOr click the magic link: %s", code, link)
	html := fmt.Sprintf(`<p>Your access code is <b>%s</b></p>
        <p>Or click <a href="%s">this link</a> to sign in directly.</p>`, code, link)

	_, err := s.Send(email, email, subject, text, html)
	return err
}
