package services

import (
	"testing"
	"time"
)

func TestTOTPValidationWindow(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Unix(1710000000, 0)
	code := totpCode(secret, now.Unix()/30)
	if len(code) != 6 {
		t.Fatalf("code length = %d, want 6", len(code))
	}
	if !ValidateTOTP(secret, code, now) {
		t.Fatal("current TOTP code was rejected")
	}
	if ValidateTOTP(secret, "000000", now) && code != "000000" {
		t.Fatal("invalid TOTP code was accepted")
	}
	if ValidateTOTP(secret, code, now.Add(2*time.Minute)) {
		t.Fatal("expired TOTP code was accepted")
	}
}

func TestSecretEncryptionRoundTrip(t *testing.T) {
	encrypted, err := EncryptSecret("sensitive-value", "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "sensitive-value" {
		t.Fatal("secret was stored as plaintext")
	}
	if got := DecryptSecret(encrypted, "test-key"); got != "sensitive-value" {
		t.Fatalf("decrypted secret = %q", got)
	}
}
