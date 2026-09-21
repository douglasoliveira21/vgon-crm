package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var ErrCaptchaFailed = errors.New("captcha_failed")

var captchaClient = &http.Client{Timeout: 10 * time.Second}

// VerifyTurnstile validates a Cloudflare Turnstile token server-side. Tokens
// are single-use and expire after a few minutes, so every login attempt needs
// a fresh one from the widget.
func VerifyTurnstile(secret, token, remoteIP string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrCaptchaFailed
	}

	form := url.Values{
		"secret":   {secret},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	resp, err := captchaClient.PostForm(turnstileVerifyURL, form)
	if err != nil {
		return fmt.Errorf("turnstile verify request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("turnstile verify response invalid: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("%w: %s", ErrCaptchaFailed, strings.Join(result.ErrorCodes, ","))
	}
	return nil
}
