package handlers

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func MediaProxy(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		messageID := c.Params("messageId")
		expires, err := strconv.ParseInt(c.Query("expires"), 10, 64)
		if err != nil || expires < time.Now().Unix() || expires > time.Now().Add(10*time.Minute).Unix() {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Link de mídia inválido ou expirado"})
		}
		companyID := c.Query("company")
		expected := mediaSignature(messageID, companyID, expires, svc.Config.JWTSecret)
		if !hmac.Equal([]byte(expected), []byte(c.Query("signature"))) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Assinatura de mídia inválida"})
		}

		// Get message media URL and instance
		var mediaURL, messageType string
		var instanceName sql.NullString
		err = svc.DB.QueryRow(`
			SELECT m.media_url, wi.instance_name, m.message_type
			FROM messages m
			JOIN conversations conv ON m.conversation_id = conv.id
			LEFT JOIN channels ch ON conv.channel_id = ch.id
			LEFT JOIN whatsapp_instances wi ON wi.channel_id = ch.id
			WHERE m.id = $1 AND m.company_id = $2 AND m.media_url IS NOT NULL
		`, messageID, companyID).Scan(&mediaURL, &instanceName, &messageType)

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Media not found"})
		}
		if strings.HasPrefix(mediaURL, "/uploads/") {
			fileName := filepath.Base(mediaURL)
			filePath := filepath.Join("/app/uploads", fileName)
			data, readErr := os.ReadFile(filePath)
			if readErr != nil || len(data) > 25*1024*1024 {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Mídia não encontrada"})
			}
			if err := scanMedia(data, svc.Config.ClamAVAddr); err != nil {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "Arquivo bloqueado pela verificação de segurança"})
			}
			c.Set("Content-Type", getMimeType(messageType))
			c.Set("Cache-Control", "private, max-age=300")
			return c.Send(data)
		}

		// Try to get media via Evolution API's getBase64FromMediaMessage
		var mediaData, mimeType string
		if instanceName.Valid {
			mediaData, mimeType, err = svc.Evolution.GetMediaBase64(instanceName.String, messageID)
		} else {
			err = fmt.Errorf("instância de mídia indisponível")
		}
		if err != nil {
			log.Printf("[MEDIA] Failed to get media via Evolution API for %s: %v", messageID, err)

			// Fallback: try to fetch the media URL directly
			if !allowedMediaURL(mediaURL, svc.Config.EvolutionAPIURL) {
				return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Origem da mídia não permitida"})
			}
			client := &http.Client{
				Timeout: 30 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) >= 3 || !allowedMediaURL(req.URL.String(), svc.Config.EvolutionAPIURL) {
						return http.ErrUseLastResponse
					}
					return nil
				},
			}
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
			if resp.ContentLength > 25*1024*1024 {
				return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "Mídia excede o limite permitido"})
			}
			c.Set("Cache-Control", "private, max-age=300")

			body, err := io.ReadAll(io.LimitReader(resp.Body, 25*1024*1024+1))
			if err != nil || len(body) > 25*1024*1024 {
				return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "Mídia excede o limite permitido"})
			}
			if err := scanMedia(body, svc.Config.ClamAVAddr); err != nil {
				log.Printf("[MEDIA] Malware scan rejected %s: %v", messageID, err)
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "Arquivo bloqueado pela verificação de segurança"})
			}
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
		if len(decoded) > 25*1024*1024 {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "Mídia excede o limite permitido"})
		}
		if err := scanMedia(decoded, svc.Config.ClamAVAddr); err != nil {
			log.Printf("[MEDIA] Malware scan rejected %s: %v", messageID, err)
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "Arquivo bloqueado pela verificação de segurança"})
		}
		c.Set("Cache-Control", "private, max-age=300")

		return c.Send(decoded)
	}
}

func signedMediaPath(messageID, companyID, secret string, expiresAt time.Time) string {
	expires := expiresAt.Unix()
	return fmt.Sprintf("/api/media/%s?company=%s&expires=%d&signature=%s",
		messageID, url.QueryEscape(companyID), expires, mediaSignature(messageID, companyID, expires, secret))
}

func signedUploadURL(baseURL, fileName, secret string, expiresAt time.Time) string {
	expires := expiresAt.Unix()
	signature := uploadSignature(fileName, expires, secret)
	return fmt.Sprintf("%s/uploads/%s?expires=%d&signature=%s",
		strings.TrimRight(baseURL, "/"), url.PathEscape(filepath.Base(fileName)), expires, signature)
}

func uploadSignature(fileName string, expires int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%s:%d", filepath.Base(fileName), expires)
	return hex.EncodeToString(mac.Sum(nil))
}

func ServeUpload(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requested := strings.TrimPrefix(c.Params("*"), "/")
		isAvatar := strings.HasPrefix(requested, "avatars/")
		if isAvatar {
			requested = strings.TrimPrefix(requested, "avatars/")
		}
		fileName := filepath.Base(requested)
		if requested == "" || requested != fileName {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if !isAvatar {
			expires, err := strconv.ParseInt(c.Query("expires"), 10, 64)
			if err != nil || expires < time.Now().Unix() || expires > time.Now().Add(15*time.Minute).Unix() ||
				!hmac.Equal([]byte(uploadSignature(fileName, expires, svc.Config.JWTSecret)), []byte(c.Query("signature"))) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Link de arquivo inválido ou expirado"})
			}
		}
		root := "/app/uploads"
		if isAvatar {
			root = "/app/uploads/avatars"
		}
		path := filepath.Join(root, fileName)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > 25*1024*1024 {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if err := scanMedia(data, svc.Config.ClamAVAddr); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "Arquivo bloqueado"})
		}
		c.Set("Cache-Control", "private, max-age=300")
		return c.SendFile(path)
	}
}

func mediaSignature(messageID, companyID string, expires int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%s:%s:%d", messageID, companyID, expires)
	return hex.EncodeToString(mac.Sum(nil))
}

func scanMedia(data []byte, address string) error {
	if strings.TrimSpace(address) == "" {
		return nil
	}
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("antivírus indisponível: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if _, err = conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	chunk := make([]byte, 32*1024)
	for {
		count, readErr := reader.Read(chunk)
		if count > 0 {
			var size [4]byte
			binary.BigEndian.PutUint32(size[:], uint32(count))
			if _, err = conn.Write(size[:]); err != nil {
				return err
			}
			if _, err = conn.Write(chunk[:count]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	_, _ = conn.Write([]byte{0, 0, 0, 0})
	response, err := bufio.NewReader(conn).ReadString(0)
	if err != nil {
		return err
	}
	if !strings.Contains(response, "OK") {
		return fmt.Errorf("ameaça detectada: %s", strings.TrimSpace(response))
	}
	return nil
}

func scanSavedUpload(fileName, address string) error {
	if strings.TrimSpace(address) == "" {
		return nil
	}
	path := filepath.Join("/app/uploads", filepath.Base(fileName))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := scanMedia(data, address); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func allowedMediaURL(rawURL, evolutionURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	evolution, _ := url.Parse(evolutionURL)
	allowedExact := map[string]bool{
		strings.ToLower(evolution.Hostname()): true,
		"mmg.whatsapp.net":                    true,
		"lookaside.fbsbx.com":                 true,
	}
	if allowedExact[host] {
		return true
	}
	for _, suffix := range []string{".whatsapp.net", ".fbcdn.net", ".fbsbx.com"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
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
	return fmt.Sprintf("%s/api/media/%s", apiURL, messageID)
}
