package domainerrors

import "errors"

var (
	EmailAlreadyTaken          = errors.New("email already taken")
	ErrInvalidOrExpiredOtp     = errors.New("Invalid or Expired Otp")
	EmailALreadyVerified       = errors.New("email already verified")
	ResendOtpCooldown          = errors.New("please wait for 1 minute before requesting a new OTP")
	EmailNotFound              = errors.New("email not found")
	InvalidEmailOrPassword     = errors.New("invalid email or password")
	VerifyEmailBeforeLoggingIN = errors.New("please verify email first")
	ErrOtpCooldownLimit        = errors.New("please wait for 1 minute before requesting a new OTP")
)
