package utils

import (
	"log"

	"github.com/junaid9001/tripneo/auth-service/config"
	"github.com/resend/resend-go/v3"
)

func SendEmail(cfg *config.Config, to, subject, body string) error {
	client := resend.NewClient(cfg.RESEND_API_KEY)

	params := &resend.SendEmailRequest{
		From:    "Tripneo <onboarding@resend.dev>",
		To:      []string{to},
		Subject: subject,
		Html:    body,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		log.Printf("failed to send email: %v", err)
		return err
	}

	log.Printf("email sent: %s", sent.Id)
	return nil
}

// body for the otp mail
var OtpBody = `
<div style="font-family:Arial,sans-serif;line-height:1.6">
  <h2>Email Verification</h2>
  <p>Your OTP code is:</p>

  <div style="font-size:28px;font-weight:bold;letter-spacing:6px;margin:20px 0">
    %s
  </div>

  <p>This code will expire in <strong>5 minutes</strong>.</p>
  <p>If you didn't request this, you can ignore this email.</p>

  <hr>
  <small>Tripneo</small>
</div>
`
