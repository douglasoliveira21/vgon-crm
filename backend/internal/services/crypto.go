package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"
)

const encryptedPrefix = "enc:"

func secretKey(seed string) []byte {
	if seed == "" {
		seed = "evocrm-local-secret"
	}
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}

func EncryptSecret(value, seed string) (string, error) {
	if value == "" || strings.HasPrefix(value, encryptedPrefix) {
		return value, nil
	}

	block, err := aes.NewCipher(secretKey(seed))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptSecret(value, seed string) string {
	if value == "" || !strings.HasPrefix(value, encryptedPrefix) {
		return value
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encryptedPrefix))
	if err != nil {
		return ""
	}

	block, err := aes.NewCipher(secretKey(seed))
	if err != nil {
		return ""
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil || len(raw) < gcm.NonceSize() {
		return ""
	}

	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}
	return string(plaintext)
}

func StringsJSON(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, `"`+strings.ReplaceAll(value, `"`, `\"`)+`"`)
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}
