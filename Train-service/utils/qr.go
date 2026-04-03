package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// QRPayload is the signed payload embedded in each QR code.
type QRPayload struct {
	BookingID  string   `json:"booking_id"`
	PNR        string   `json:"pnr"`
	UserID     string   `json:"user_id"`
	TrainName  string   `json:"train_name"`
	TrainNum   string   `json:"train_number"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Departure  string   `json:"departure"`
	Class      string   `json:"class"`
	Passengers []string `json:"passengers"`
	Berths     []string `json:"berths"`
	IssuedAt   string   `json:"issued_at"`
}

// GenerateQRPayload builds an HMAC-SHA256 signed, base64-encoded QR token string.
// This token is stored in train_tickets.qr_data and encoded into the QR image.
func GenerateQRPayload(
	bookingID, pnr, userID, trainName, trainNum,
	from, to string,
	departure time.Time,
	class string,
	passengers, berths []string,
) string {
	payload := QRPayload{
		BookingID:  bookingID,
		PNR:        pnr,
		UserID:     userID,
		TrainName:  trainName,
		TrainNum:   trainNum,
		From:       from,
		To:         to,
		Departure:  departure.Format(time.RFC3339),
		Class:      class,
		Passengers: passengers,
		Berths:     berths,
		IssuedAt:   time.Now().Format(time.RFC3339),
	}

	payloadBytes, _ := json.Marshal(payload)
	sig := computeHMAC(payloadBytes)

	envelope := map[string]string{
		"payload":   base64.StdEncoding.EncodeToString(payloadBytes),
		"signature": sig,
	}
	envelopeBytes, _ := json.Marshal(envelope)
	return base64.StdEncoding.EncodeToString(envelopeBytes)
}

// VerifyQRToken decodes and validates an HMAC-signed QR token.
// Returns (bookingID, true) on success, ("", false) on tamper/invalid.
func VerifyQRToken(qrData, expectedBookingID string) bool {
	// Decode outer envelope
	envelopeBytes, err := base64.StdEncoding.DecodeString(qrData)
	if err != nil {
		return false
	}
	var envelope map[string]string
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		return false
	}

	payloadB64, ok1 := envelope["payload"]
	sig, ok2 := envelope["signature"]
	if !ok1 || !ok2 {
		return false
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		return false
	}

	// Verify HMAC signature
	expectedSig := computeHMAC(payloadBytes)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return false
	}

	// Decode payload and check booking_id
	var payload QRPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false
	}
	return payload.BookingID == expectedBookingID
}

// GenerateQRToken produces a simple HMAC token string for a booking — used by the
// old VerifyTicket handler that sends (bookingID, token) separately.
func GenerateQRToken(bookingID string) string {
	secret := getHMACSecret()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(bookingID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyQRTokenSimple is the handler-compatible version: verifies (bookingID, token).
func VerifyQRTokenSimple(bookingID, token string) bool {
	if token == "" {
		return false
	}
	expected := GenerateQRToken(bookingID)
	return hmac.Equal([]byte(expected), []byte(token))
}

func computeHMAC(data []byte) string {
	secret := getHMACSecret()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func getHMACSecret() string {
	s := os.Getenv("HMAC_SECRET")
	if s == "" {
		return "default-hmac-secret-change-in-production"
	}
	return s
}
