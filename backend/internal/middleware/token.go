package middleware

import (
	"fmt"

	"github.com/evocrm/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// ValidateRefreshToken validates a refresh token and returns the parsed token
func ValidateRefreshToken(tokenString string, cfg *config.Config) (*jwt.Token, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTRefreshSecret), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}
