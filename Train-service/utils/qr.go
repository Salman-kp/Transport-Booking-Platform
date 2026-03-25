package utils

// VerifyQRToken validates an HMAC-signed QR token.
// Full implementation in Phase 7 when QR generation is built.
func VerifyQRToken(bookingID, token string) bool {
	return token != ""
}
