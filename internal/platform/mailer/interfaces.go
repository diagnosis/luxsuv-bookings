package mailer

type Service interface {
	Send(toEmail, toName, subject, text, html string) (string, error)
	SendGuestAccess(email, code, link string) error
}
