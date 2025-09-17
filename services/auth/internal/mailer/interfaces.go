package mailer

type Service interface {
	SendVerificationEmail(toEmail, toName, verifyURL, token string) error
	SendGuestAccessEmail(email, code, magicLink string) error
}