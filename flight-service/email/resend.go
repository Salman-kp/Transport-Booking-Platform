package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type ResendClient struct {
	apiKey string
	from   string
	client *http.Client
}

func NewResendClient(apiKey string, fromAddress string) *ResendClient {
	if apiKey == "" {
		log.Println("[EMAIL] RESEND_API_KEY not set — email sending disabled")
		return nil
	}
	if fromAddress == "" {
		fromAddress = "TripNEO <bookings@tripneo.in>"
	}
	return &ResendClient{
		apiKey: apiKey,
		from:   fromAddress,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type BookingEmailData struct {
	ToEmail       string
	ToName        string
	PNR           string
	FlightNumber  string
	Origin        string
	Destination   string
	DepartureTime string
	ArrivalTime   string
	SeatClass     string
	TotalAmount   float64
	Currency      string
	Passengers    []string
}

func (r *ResendClient) SendBookingConfirmation(data BookingEmailData) error {
	if r == nil {
		return nil
	}

	subject := fmt.Sprintf("✈️ Booking Confirmed — %s | PNR: %s", data.FlightNumber, data.PNR)
	html := buildConfirmationHTML(data)

	payload := map[string]interface{}{
		"from":    r.from,
		"to":      []string{data.ToEmail},
		"subject": subject,
		"html":    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[EMAIL] Confirmation sent to %s for PNR %s", data.ToEmail, data.PNR)
	return nil
}

func buildConfirmationHTML(data BookingEmailData) string {
	passengerRows := ""
	for i, name := range data.Passengers {
		passengerRows += fmt.Sprintf(`<tr><td style="padding:8px 12px;border-bottom:1px solid #f0f0f0;color:#666;">%d</td><td style="padding:8px 12px;border-bottom:1px solid #f0f0f0;color:#1a1a1a;font-weight:500;">%s</td></tr>`, i+1, name)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background-color:#f4f4f5;font-family:'Segoe UI',Roboto,sans-serif;">
<div style="max-width:600px;margin:0 auto;background:#ffffff;">

  <div style="background:#1a1a1a;padding:32px 24px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;letter-spacing:2px;">TRIPNEO</h1>
    <p style="color:#fcc54c;margin:8px 0 0;font-size:11px;letter-spacing:3px;text-transform:uppercase;">Booking Confirmed</p>
  </div>

  <div style="padding:32px 24px;">
    <div style="background:#f8fdf8;border:1px solid #d4edda;border-radius:8px;padding:16px;margin-bottom:24px;text-align:center;">
      <span style="color:#28a745;font-size:20px;">✓</span>
      <p style="color:#155724;margin:8px 0 0;font-weight:600;">Your flight has been booked successfully!</p>
    </div>

    <div style="background:#fafafa;border-radius:8px;padding:20px;margin-bottom:24px;">
      <table style="width:100%%;border-collapse:collapse;">
        <tr>
          <td style="padding:6px 0;color:#999;font-size:11px;text-transform:uppercase;letter-spacing:1px;">PNR</td>
          <td style="padding:6px 0;text-align:right;font-size:20px;font-weight:700;color:#1a1a1a;letter-spacing:2px;">%s</td>
        </tr>
        <tr>
          <td style="padding:6px 0;color:#999;font-size:11px;text-transform:uppercase;letter-spacing:1px;">Flight</td>
          <td style="padding:6px 0;text-align:right;font-weight:600;color:#1a1a1a;">%s</td>
        </tr>
        <tr>
          <td style="padding:6px 0;color:#999;font-size:11px;text-transform:uppercase;letter-spacing:1px;">Class</td>
          <td style="padding:6px 0;text-align:right;font-weight:500;color:#1a1a1a;">%s</td>
        </tr>
      </table>
    </div>

    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:24px;padding:20px;background:#fafafa;border-radius:8px;">
      <div style="text-align:center;flex:1;">
        <p style="margin:0;font-size:28px;font-weight:700;color:#1a1a1a;">%s</p>
        <p style="margin:4px 0 0;color:#999;font-size:12px;">%s</p>
      </div>
      <div style="flex:1;text-align:center;color:#ccc;">✈️</div>
      <div style="text-align:center;flex:1;">
        <p style="margin:0;font-size:28px;font-weight:700;color:#1a1a1a;">%s</p>
        <p style="margin:4px 0 0;color:#999;font-size:12px;">%s</p>
      </div>
    </div>

    <div style="margin-bottom:24px;">
      <p style="color:#999;font-size:11px;text-transform:uppercase;letter-spacing:1px;margin-bottom:8px;">Passengers</p>
      <table style="width:100%%;border-collapse:collapse;">%s</table>
    </div>

    <div style="background:#1a1a1a;border-radius:8px;padding:20px;text-align:center;">
      <p style="color:#999;font-size:11px;text-transform:uppercase;letter-spacing:1px;margin:0 0 4px;">Total Paid</p>
      <p style="color:#fcc54c;font-size:28px;font-weight:700;margin:0;">%s %.2f</p>
    </div>
  </div>

  <div style="padding:24px;text-align:center;border-top:1px solid #f0f0f0;">
    <p style="color:#999;font-size:11px;margin:0;">This is an automated confirmation from TripNEO.</p>
    <p style="color:#999;font-size:11px;margin:4px 0 0;">For support, contact us at support@tripneo.in</p>
  </div>

</div>
</body>
</html>`,
		data.PNR,
		data.FlightNumber,
		data.SeatClass,
		data.Origin,
		data.DepartureTime,
		data.Destination,
		data.ArrivalTime,
		passengerRows,
		data.Currency,
		data.TotalAmount,
	)
}
