package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func MediaProxy(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		messageID := c.Params("messageId")

		// Get message media URL and instance
		var mediaURL, instanceName, messageType string
		err := svc.DB.QueryRow(`
			SELECT m.media_url, wi.instance_name, m.message_type
			FROM messages m
			JOIN conversations conv ON m.conversation_id = conv.id
			JOIN channels ch ON conv.channel_id = ch.id
			JOIN whatsapp_instances wi ON wi.channel_id = ch.id
			WHERE m.id = $1 AND m.media_url IS NOT NULL
		`, messageID).Scan(&mediaURL, &instanceName, &messageType)

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Media not found"})
		}

		// Try to get media via Evolution API's getBase64FromMediaMessage
		mediaData, mimeType, err := svc.Evolution.GetMediaBase64(instanceName, messageID)
		if err != nil {
			log.Printf("[MEDIA] Failed to get media via Evolution API for %s: %v", messageID, err)

			// Fallback: try to fetch the media URL directly
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Get(mediaURL)
			if err != nil {
				return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to fetch media"})
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Media unavailable"})
			}

			contentType := resp.Header.Get("Content-Type")
			if contentType == "" {
				contentType = getMimeType(messageType)
			}

			c.Set("Content-Type", contentType)
			c.Set("Cache-Control", "public, max-age=86400")

			body, _ := io.ReadAll(resp.Body)
			return c.Send(body)
		}

		// Decode base64 and serve
		decoded, err := base64.StdEncoding.DecodeString(mediaData)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode media"})
		}

		if mimeType == "" {
			mimeType = getMimeType(messageType)
		}

		c.Set("Content-Type", mimeType)
		c.Set("Cache-Control", "public, max-age=86400")

		return c.Send(decoded)
	}
}

func getMimeType(messageType string) string {
	switch messageType {
	case "audio":
		return "audio/ogg"
	case "image":
		return "image/jpeg"
	case "gif", "sticker":
		return "image/gif"
	case "video":
		return "video/mp4"
	case "document":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

// GetMediaURL returns the proxied media URL for a message
func GetMediaURL(apiURL, messageID string) string {
	return fmt.Sprintf("%s/media/%s", apiURL, messageID)
}
