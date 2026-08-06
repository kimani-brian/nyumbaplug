// Package mail sends transactional emails (currently OTP codes) via Resend.
package mail

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/resend/resend-go/v3"
)

// Sender delivers an OTP email to a recipient. Implementations must be safe
// for concurrent use.
type Sender interface {
	SendOTP(ctx context.Context, toEmail, code string) error
}

// ResendSender sends emails through the Resend API.
type ResendSender struct {
	client    *resend.Client
	fromEmail string
}

// NewResendSender returns a Sender backed by the Resend Go SDK. If apiKey is
// empty the sender is disabled; it only logs the recipient and code so the app
// still works before Resend credentials are configured.
func NewResendSender(apiKey, fromEmail string) Sender {
	if apiKey == "" {
		return &loggingSender{}
	}
	return &ResendSender{client: resend.NewClient(apiKey), fromEmail: fromEmail}
}

func (s *ResendSender) SendOTP(ctx context.Context, toEmail, code string) error {
	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{strings.TrimSpace(toEmail)},
		Subject: "Your NyumbaPlug verification code",
		Html:    otpHtml(code),
	}
	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("resend send otp: %w", err)
	}
	return nil
}

// loggingSender is a fallback used when no Resend API key is configured. It
// never reaches the network.
type loggingSender struct{}

func (s *loggingSender) SendOTP(_ context.Context, toEmail, code string) error {
	log.Printf("[mail] OTP for %s: %s (no RESEND_API_KEY set, not sent)", toEmail, code)
	return nil
}

func otpHtml(code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
  <body style="margin:0;padding:0;background:#f5f7fa;font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;">
    <div style="max-width:480px;margin:40px auto;background:#ffffff;border:1px solid #e2e8f0;border-radius:16px;padding:32px;">
      <h2 style="margin:0 0 8px;color:#0f172a;">Verify your email</h2>
      <p style="margin:0 0 24px;color:#475569;font-size:15px;line-height:1.5;">
        Use the code below to finish signing up for NyumbaPlug. This code expires in 10 minutes.
      </p>
      <div style="text-align:center;background:#f1f5f9;border:1px dashed #cbd5e1;border-radius:12px;padding:20px;">
        <span style="font-size:32px;font-weight:700;letter-spacing:8px;color:#0f766e;">%s</span>
      </div>
      <p style="margin:24px 0 0;color:#64748b;font-size:13px;line-height:1.5;">
        If you didn't request this code, you can safely ignore this email.
      </p>
    </div>
  </body>
</html>`, code)
}
