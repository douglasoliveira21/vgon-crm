package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net/smtp"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/middleware"
	"github.com/evocrm/backend/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db  *sql.DB
	cfg *config.Config
}

func NewAuthService(db *sql.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	CompanyName string `json:"company_name"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *models.User `json:"user"`
}

func (s *AuthService) Login(req *LoginRequest) (*AuthResponse, error) {
	var user models.User
	var roleSlug sql.NullString

	err := s.db.QueryRow(`
		SELECT u.id, u.company_id, u.name, u.email, u.password_hash, u.avatar_url, 
			   u.is_active, u.is_online, COALESCE(u.availability_status, 'offline'),
			   COALESCE(u.is_super_admin, false), r.slug, r.name
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1 AND u.is_active = true
	`, strings.ToLower(req.Email)).Scan(
		&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.PasswordHash,
		&user.AvatarURL, &user.IsActive, &user.IsOnline, &user.AvailabilityStatus,
		&user.IsSuperAdmin, &roleSlug, &user.RoleName,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	user.RoleSlug = ""
	if roleSlug.Valid {
		user.RoleSlug = roleSlug.String
	}

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokens(
		user.ID, user.CompanyID, user.RoleSlug, user.Email, s.cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Save refresh token (hashed)
	hashedRefresh := hashToken(refreshToken)
	_, err = s.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), user.ID, hashedRefresh, time.Now().Add(s.cfg.JWTRefreshExpiry))
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	// Update presence
	user.IsOnline = true
	user.AvailabilityStatus = "online"
	s.db.Exec("UPDATE users SET is_online = true, availability_status = 'online', last_seen_at = NOW() WHERE id = $1", user.ID)

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

func (s *AuthService) Register(req *RegisterRequest) (*AuthResponse, error) {
	// Check if email already exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", strings.ToLower(req.Email)).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create company
	companyID := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.CompanyName, " ", "-"))

	_, err = s.db.Exec(`
		INSERT INTO companies (id, name, slug) VALUES ($1, $2, $3)
	`, companyID, req.CompanyName, slug+"-"+companyID[:8])
	if err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	// Get admin role
	var roleID string
	err = s.db.QueryRow("SELECT id FROM roles WHERE slug = 'admin' AND is_system = true LIMIT 1").Scan(&roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Create user
	userID := uuid.New().String()
	_, err = s.db.Exec(`
		INSERT INTO users (id, company_id, role_id, name, email, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, companyID, roleID, req.Name, strings.ToLower(req.Email), string(hashedPassword))
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokens(userID, companyID, "admin", req.Email, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Save refresh token (hashed)
	s.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), userID, hashToken(refreshToken), time.Now().Add(s.cfg.JWTRefreshExpiry))

	user := &models.User{
		ID:        userID,
		CompanyID: companyID,
		Name:      req.Name,
		Email:     strings.ToLower(req.Email),
		IsActive:  true,
		RoleSlug:  "admin",
		RoleName:  "Administrador",
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) RefreshToken(refreshTokenStr string) (*AuthResponse, error) {
	// Validate refresh token
	claims := &middleware.Claims{}
	token, err := middleware.ValidateRefreshToken(refreshTokenStr, s.cfg)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token")
	}

	claims = token.Claims.(*middleware.Claims)

	// Check if token exists in DB (compare hashed)
	hashedRefresh := hashToken(refreshTokenStr)
	var tokenID string
	err = s.db.QueryRow(`
		SELECT id FROM refresh_tokens WHERE token = $1 AND user_id = $2 AND expires_at > NOW()
	`, hashedRefresh, claims.UserID).Scan(&tokenID)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Delete old refresh token
	s.db.Exec("DELETE FROM refresh_tokens WHERE id = $1", tokenID)

	// Generate new tokens
	accessToken, newRefreshToken, err := middleware.GenerateTokens(
		claims.UserID, claims.CompanyID, claims.RoleSlug, claims.Email, s.cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Save new refresh token (hashed)
	s.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), claims.UserID, hashToken(newRefreshToken), time.Now().Add(s.cfg.JWTRefreshExpiry))

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) RequestPasswordReset(req *ForgotPasswordRequest) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return fmt.Errorf("email is required")
	}

	var userID, name, userEmail string
	err := s.db.QueryRow(`
		SELECT id, name, email
		FROM users
		WHERE LOWER(email) = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`, email).Scan(&userID, &name, &userEmail)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	if s.cfg.SMTPHost == "" || s.cfg.SMTPUser == "" || s.cfg.SMTPPass == "" || s.cfg.SMTPFrom == "" {
		return fmt.Errorf("smtp is not configured")
	}

	token, tokenHash, err := generatePasswordResetToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate old reset tokens: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), userID, tokenHash, time.Now().Add(time.Hour))
	if err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reset token: %w", err)
	}

	resetLink := strings.TrimRight(s.cfg.FrontendURL, "/") + "/reset-password?token=" + token
	if err := s.sendPasswordResetEmail(userEmail, name, resetLink); err != nil {
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	return nil
}

func (s *AuthService) ResetPassword(req *ResetPasswordRequest) error {
	token := strings.TrimSpace(req.Token)
	password := req.NewPassword
	if password == "" {
		password = req.Password
	}
	if token == "" || password == "" {
		return fmt.Errorf("token and password are required")
	}
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	tokenHash := hashPasswordResetToken(token)
	newHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	var tokenID, userID string
	err = tx.QueryRow(`
		SELECT id, user_id
		FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		LIMIT 1
	`, tokenHash).Scan(&tokenID, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("invalid or expired reset token")
	}
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2 AND is_active = true
	`, string(newHash), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	_, err = tx.Exec("UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1", tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark reset token as used: %w", err)
	}

	_, err = tx.Exec("DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	return tx.Commit()
}

func generatePasswordResetToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(bytes)
	return token, hashPasswordResetToken(token), nil
}

func hashPasswordResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) sendPasswordResetEmail(to, name, resetLink string) error {
	host := s.cfg.SMTPHost
	port := s.cfg.SMTPPort
	from := s.cfg.SMTPFrom
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, host)

	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = to
	}

	subject := "Recuperacao de senha - crmvgon"
	body := fmt.Sprintf(`
		<p>Ola, %s.</p>
		<p>Recebemos uma solicitacao para redefinir sua senha no crmvgon.</p>
		<p><a href="%s">Clique aqui para criar uma nova senha</a>.</p>
		<p>Este link expira em 1 hora. Se voce nao solicitou a troca, ignore este e-mail.</p>
	`, html.EscapeString(displayName), html.EscapeString(resetLink))

	message := strings.Join([]string{
		"From: crmvgon <" + from + ">",
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	return smtp.SendMail(host+":"+port, auth, from, []string{to}, []byte(message))
}

func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(`
		SELECT u.id, u.company_id, u.name, u.email, u.avatar_url, u.phone,
			   u.is_active, u.is_online, COALESCE(u.availability_status, 'offline'),
			   COALESCE(u.is_super_admin, false), r.slug, r.name
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`, userID).Scan(
		&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.AvatarURL,
		&user.Phone, &user.IsActive, &user.IsOnline, &user.AvailabilityStatus,
		&user.IsSuperAdmin, &user.RoleSlug, &user.RoleName,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}


// hashToken creates a SHA256 hash of a token for secure storage
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
