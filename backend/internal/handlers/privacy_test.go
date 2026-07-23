package handlers

import (
	"testing"
	"time"
)

func TestUnsubscribeTokenIsSignedScopedAndExpires(t *testing.T) {
	secret := "test-secret"
	token := createUnsubscribeToken("tenant-a", "contact-a", "email", secret, time.Now().Add(time.Hour))
	claims, ok := parseUnsubscribeToken(token, secret)
	if !ok || claims.CompanyID != "tenant-a" || claims.ContactID != "contact-a" || claims.Channel != "email" {
		t.Fatal("valid unsubscribe token was rejected or lost its scope")
	}
	if _, ok := parseUnsubscribeToken(token, "other-secret"); ok {
		t.Fatal("unsubscribe token was accepted with another secret")
	}
	expired := createUnsubscribeToken("tenant-a", "contact-a", "email", secret, time.Now().Add(-time.Minute))
	if _, ok := parseUnsubscribeToken(expired, secret); ok {
		t.Fatal("expired unsubscribe token was accepted")
	}
}

func TestNormalizeSuppressionDestination(t *testing.T) {
	if got := normalizeSuppressionDestination("email", " User@Example.COM "); got != "user@example.com" {
		t.Fatalf("email normalization = %q", got)
	}
	if got := normalizeSuppressionDestination("whatsapp", "+55 (31) 99999-0000"); got != "5531999990000" {
		t.Fatalf("phone normalization = %q", got)
	}
}
