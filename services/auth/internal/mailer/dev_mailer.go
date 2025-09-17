package mailer

import (
	"fmt"

	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
)

type DevMailer struct{}

func NewDevMailer() *DevMailer {
	return &DevMailer{}
}

func (d *DevMailer) SendVerificationEmail(toEmail, toName, verifyURL, token string) error {
	logger.Info("📧 [DEV MAIL] Verification Email",
		"to", toEmail,
		"name", toName,
		"verify_url", verifyURL,
		"token", token,
	)
	
	fmt.Printf("\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"📧 VERIFICATION EMAIL (DEV MODE)\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"To: %s (%s)\n" +
		"Subject: Verify your LuxSUV account\n" +
		"\n" +
		"Verification URL: %s\n" +
		"Token: %s\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n",
		toEmail, toName, verifyURL, token)
	
	return nil
}

func (d *DevMailer) SendGuestAccessEmail(email, code, magicLink string) error {
	logger.Info("📧 [DEV MAIL] Guest Access Email",
		"to", email,
		"code", code,
		"magic_link", magicLink,
	)
	
	fmt.Printf("\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"📧 GUEST ACCESS EMAIL (DEV MODE)\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"To: %s\n" +
		"Subject: Your LuxSUV access code\n" +
		"\n" +
		"Access Code: %s\n" +
		"Magic Link: %s\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n",
		email, code, magicLink)
	
	return nil
}