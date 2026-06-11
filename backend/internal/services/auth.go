package services

import (
	"database/sql"
	"errors"
	"fmt"
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
			   u.is_active, u.is_online, r.slug, r.name
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1 AND u.is_active = true
	`, strings.ToLower(req.Email)).Scan(
		&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.PasswordHash,
		&user.AvatarURL, &user.IsActive, &user.IsOnline, &roleSlug, &user.RoleName,
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

	// Save refresh token
	_, err = s.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), user.ID, refreshToken, time.Now().Add(s.cfg.JWTRefreshExpiry))
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	// Update last seen
	s.db.Exec("UPDATE users SET is_online = true, last_seen_at = NOW() WHERE id = $1", user.ID)

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

	// Save refresh token
	s.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), userID, refreshToken, time.Now().Add(s.cfg.JWTRefreshExpiry))

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

	// Check if token exists in DB
	var tokenID string
	err = s.db.QueryRow(`
		SELECT id FROM refresh_tokens WHERE token = $1 AND user_id = $2 AND expires_at > NOW()
	`, refreshTokenStr, claims.UserID).Scan(&tokenID)
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

	// Save new refresh token
	s.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), claims.UserID, newRefreshToken, time.Now().Add(s.cfg.JWTRefreshExpiry))

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(`
		SELECT u.id, u.company_id, u.name, u.email, u.avatar_url, u.phone,
			   u.is_active, u.is_online, r.slug, r.name
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`, userID).Scan(
		&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.AvatarURL,
		&user.Phone, &user.IsActive, &user.IsOnline, &user.RoleSlug, &user.RoleName,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
