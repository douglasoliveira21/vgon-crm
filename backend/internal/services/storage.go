package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const uploadsDir = "/app/uploads"

// MediaSignature computes the HMAC that authorizes a time-limited fetch of a
// message's media through the /api/media/:messageId proxy. Shared between
// internal/handlers (REST responses, which already produced a correctly
// signed path) and internal/services (the WebSocket broadcast for a message
// that just arrived) so both hand the frontend an equally usable link —
// broadcasting the raw, still-encrypted WhatsApp CDN URL instead left a
// live-received image/audio/video/document showing as unavailable until the
// next full page load re-fetched it through the REST endpoint.
func MediaSignature(messageID, companyID string, expires int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%s:%s:%d", messageID, companyID, expires)
	return hex.EncodeToString(mac.Sum(nil))
}

// SignedMediaPath builds the signed relative path the frontend uses to fetch
// a message's media through the backend's proxy/decrypt layer.
func SignedMediaPath(messageID, companyID, secret string, expiresAt time.Time) string {
	expires := expiresAt.Unix()
	return fmt.Sprintf("/api/media/%s?company=%s&expires=%d&signature=%s",
		messageID, url.QueryEscape(companyID), expires, MediaSignature(messageID, companyID, expires, secret))
}

func init() {
	// Ensure uploads directory exists on startup
	os.MkdirAll(uploadsDir, 0777)
}

// SaveBase64File saves a base64 encoded file to disk and returns the file path
func SaveBase64File(base64Data, fileExtension string) (string, error) {
	// Ensure uploads directory exists
	if err := os.MkdirAll(uploadsDir, 0777); err != nil {
		return "", fmt.Errorf("failed to create uploads dir: %w", err)
	}

	// Remove data URI prefix if present (data:audio/ogg;base64,...)
	rawBase64 := base64Data
	if idx := strings.Index(base64Data, ","); idx != -1 {
		rawBase64 = base64Data[idx+1:]
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(rawBase64)
	if err != nil {
		// Try with padding fix
		rawBase64 = strings.TrimRight(rawBase64, "=")
		padding := 4 - len(rawBase64)%4
		if padding < 4 {
			rawBase64 += strings.Repeat("=", padding)
		}
		decoded, err = base64.StdEncoding.DecodeString(rawBase64)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64: %w", err)
		}
	}
	if len(decoded) > 25*1024*1024 {
		return "", fmt.Errorf("file exceeds the 25MB limit")
	}

	// Generate unique filename
	fileName := uuid.New().String() + fileExtension
	filePath := filepath.Join(uploadsDir, fileName)

	// Write file
	if err := os.WriteFile(filePath, decoded, 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	log.Printf("[STORAGE] Saved file: %s (%d bytes)", filePath, len(decoded))

	return fileName, nil
}

// GetExtensionFromMime returns file extension from data URI or media type
func GetExtensionFromBase64(base64Data string) string {
	if strings.HasPrefix(base64Data, "data:") {
		// Extract mime from data:audio/ogg;base64,...
		mimeEnd := strings.Index(base64Data, ";")
		if mimeEnd > 5 {
			mime := base64Data[5:mimeEnd]
			return MimeToExtension(mime)
		}
	}
	return ""
}

// GetExtensionFromType returns extension based on media type
func GetExtensionFromType(mediaType string) string {
	switch mediaType {
	case "audio":
		return ".ogg"
	case "image":
		return ".jpg"
	case "video":
		return ".mp4"
	case "document":
		return ".pdf"
	default:
		return ".bin"
	}
}

// MimeToExtension converts MIME type to file extension
func MimeToExtension(mime string) string {
	switch mime {
	case "audio/ogg", "audio/ogg; codecs=opus":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/webm":
		return ".webm"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	case "application/msword":
		return ".doc"
	default:
		return ".bin"
	}
}
