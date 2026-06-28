package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type emailOAuthState struct {
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
	Provider  string `json:"provider"`
	ExpiresAt int64  `json:"expires_at"`
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func StartEmailOAuth(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider := strings.ToLower(c.Params("provider"))
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		redirectURI := emailOAuthRedirectURI(svc, provider)
		state, err := signEmailOAuthState(svc.Config.JWTSecret, emailOAuthState{
			CompanyID: companyID,
			UserID:    userID,
			Provider:  provider,
			ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create OAuth state"})
		}

		authURL, err := buildEmailOAuthURL(svc, provider, redirectURI, state)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"auth_url": authURL})
	}
}

func EmailOAuthCallback(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider := strings.ToLower(c.Params("provider"))
		code := c.Query("code")
		stateValue := c.Query("state")
		if code == "" || stateValue == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Autorização inválida")
		}

		state, err := verifyEmailOAuthState(svc.Config.JWTSecret, stateValue)
		if err != nil || state.Provider != provider || time.Now().Unix() > state.ExpiresAt {
			return c.Status(fiber.StatusBadRequest).SendString("Estado OAuth inválido ou expirado")
		}

		redirectURI := emailOAuthRedirectURI(svc, provider)
		token, err := exchangeEmailOAuthCode(svc, provider, code, redirectURI)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		emailAddress, err := fetchOAuthAccountEmail(provider, token.AccessToken)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		channelID := uuid.New().String()
		name := "Gmail"
		if provider == "outlook" {
			name = "Outlook"
		}
		if emailAddress != "" {
			name += " - " + emailAddress
		}
		settings := services.EmailChannelSettings{
			Provider:     provider,
			Username:     emailAddress,
			EmailAddress: emailAddress,
			Mailbox:      "INBOX",
			UseTLS:       true,
			MaxImport:    100,
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenExpiry:  time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
		}
		settingsJSON, _ := json.Marshal(settings)
		_, err = svc.DB.Exec(`
			INSERT INTO channels (id, company_id, name, type, status, settings, is_active)
			VALUES ($1, $2, $3, 'email', 'connected', $4, true)
		`, channelID, state.CompanyID, name, settingsJSON)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		_, _ = svc.Email.SyncChannel(state.CompanyID, channelID)
		redirect := strings.TrimRight(svc.Config.FrontendURL, "/") + "/channels?tab=email"
		return c.Redirect(redirect, fiber.StatusFound)
	}
}

func buildEmailOAuthURL(svc *services.Container, provider, redirectURI, state string) (string, error) {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("redirect_uri", redirectURI)
	values.Set("state", state)

	if provider == "gmail" {
		if svc.Config.GoogleClientID == "" {
			return "", fmt.Errorf("GOOGLE_CLIENT_ID não configurado")
		}
		values.Set("client_id", svc.Config.GoogleClientID)
		values.Set("access_type", "offline")
		values.Set("prompt", "consent")
		values.Set("scope", "openid email https://www.googleapis.com/auth/gmail.readonly")
		return "https://accounts.google.com/o/oauth2/v2/auth?" + values.Encode(), nil
	}
	if provider == "outlook" {
		if svc.Config.MicrosoftClientID == "" {
			return "", fmt.Errorf("MICROSOFT_CLIENT_ID não configurado")
		}
		values.Set("client_id", svc.Config.MicrosoftClientID)
		values.Set("response_mode", "query")
		values.Set("scope", "offline_access User.Read Mail.Read")
		return "https://login.microsoftonline.com/common/oauth2/v2.0/authorize?" + values.Encode(), nil
	}
	return "", fmt.Errorf("provedor não suportado")
}

func exchangeEmailOAuthCode(svc *services.Container, provider, code, redirectURI string) (oauthTokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", redirectURI)

	tokenURL := ""
	if provider == "gmail" {
		tokenURL = "https://oauth2.googleapis.com/token"
		values.Set("client_id", svc.Config.GoogleClientID)
		values.Set("client_secret", svc.Config.GoogleClientSecret)
	} else if provider == "outlook" {
		tokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
		values.Set("client_id", svc.Config.MicrosoftClientID)
		values.Set("client_secret", svc.Config.MicrosoftClientSecret)
	}

	resp, err := http.PostForm(tokenURL, values)
	if err != nil {
		return oauthTokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return oauthTokenResponse{}, fmt.Errorf("falha ao trocar código OAuth: %s", string(body))
	}
	var token oauthTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&token)
	return token, err
}

func fetchOAuthAccountEmail(provider, accessToken string) (string, error) {
	endpoint := "https://www.googleapis.com/oauth2/v2/userinfo"
	if provider == "outlook" {
		endpoint = "https://graph.microsoft.com/v1.0/me?$select=mail,userPrincipalName"
	}
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("falha ao buscar conta autorizada: %s", string(body))
	}
	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", err
	}
	if provider == "gmail" {
		if email, _ := raw["email"].(string); email != "" {
			return strings.ToLower(email), nil
		}
	}
	if mailValue, _ := raw["mail"].(string); mailValue != "" {
		return strings.ToLower(mailValue), nil
	}
	if upn, _ := raw["userPrincipalName"].(string); upn != "" {
		return strings.ToLower(upn), nil
	}
	return "", fmt.Errorf("não foi possível identificar o e-mail autorizado")
}

func emailOAuthRedirectURI(svc *services.Container, provider string) string {
	base := strings.TrimRight(svc.Config.OAuthRedirectBaseURL, "/")
	if base == "" {
		base = strings.TrimRight(svc.Config.EvolutionWebhookURL, "/")
		base = strings.TrimSuffix(base, "/api/webhooks/evolution")
	}
	return base + "/api/oauth/email/" + provider + "/callback"
}

func signEmailOAuthState(secret string, state emailOAuthState) (string, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadPart))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadPart + "." + signature, nil
}

func verifyEmailOAuthState(secret, value string) (emailOAuthState, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return emailOAuthState{}, fmt.Errorf("invalid state")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return emailOAuthState{}, fmt.Errorf("invalid state signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return emailOAuthState{}, err
	}
	var state emailOAuthState
	err = json.Unmarshal(payload, &state)
	return state, err
}
