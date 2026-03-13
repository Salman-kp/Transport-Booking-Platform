package utils

import "testing"

func TestPasswordhHashingAndComparing(t *testing.T) {
	plain := "helloworld"
	hasehd, err := GenerateHashedPassword(plain)
	if err != nil {
		t.Fatalf("hashing password failed ,err: %v", err)
	}
	if hasehd == plain {
		t.Fatalf("hashing password failed, both input and output is the same")
	}
	if hasehd == "" {
		t.Fatalf("hashing returned plain string")
	}

	err = ValidatePassword(hasehd, plain)
	if err != nil {
		t.Fatalf("comparing hashed and plain shouldnt return error")
	}
}

func TestOtp(t *testing.T) {
	otp := GenerateOtp()
	if len(otp) != 5 {
		t.Fatalf("otp shouldnt be >or< 6")
	}
	if otp == "" {
		t.Fatalf("otp shouldnt be empty")
	}
}
