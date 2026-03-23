package utils

// VerifyQRToken validates an HMAC-signed QR token.
// Full implementation in Phase 7 when QR generation is built.
func VerifyQRToken(bookingID, token string) bool {
	// TODO: Phase 7 — implement HMAC-SHA256 verification
	return token != ""
}
