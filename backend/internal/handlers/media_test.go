package handlers

import (
	"strings"
	"testing"
	"time"
)

func TestSignedMediaPath(t *testing.T) {
	expiresAt := time.Now().Add(5 * time.Minute).Truncate(time.Second)
	path := signedMediaPath("message-id", "company-id", "secret", expiresAt)
	if !strings.Contains(path, "signature=") || !strings.Contains(path, "expires=") {
		t.Fatalf("signed path is incomplete: %s", path)
	}
	expected := mediaSignature("message-id", "company-id", expiresAt.Unix(), "secret")
	if !strings.Contains(path, expected) {
		t.Fatal("signed path does not contain the expected signature")
	}
	if expected == mediaSignature("message-id", "other-company", expiresAt.Unix(), "secret") {
		t.Fatal("signature is not bound to the tenant")
	}
}

func TestAllowedMediaURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://mmg.whatsapp.net/file", true},
		{"https://cdn.fbcdn.net/file", true},
		{"https://evolution.example.com/media", true},
		{"http://mmg.whatsapp.net/file", false},
		{"https://evil.example/file", false},
		{"file:///etc/passwd", false},
	}
	for _, test := range tests {
		if got := allowedMediaURL(test.url, "https://evolution.example.com"); got != test.want {
			t.Errorf("allowedMediaURL(%q) = %v, want %v", test.url, got, test.want)
		}
	}
}
