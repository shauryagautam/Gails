package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("change_me_in_production")

type contextKey string

const userIDContextKey contextKey = "gails_user_id"

// InitJWT sets the JWT secret key.
func InitJWT(secret string) {
	if secret != "" {
		secretKey = []byte(secret)
	}
}

// GenerateToken creates a JWT token for the given user ID.
func GenerateToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	return token.SignedString(secretKey)
}

// ParseToken validates a JWT token and returns the user ID.
func ParseToken(tokenStr string) (uint, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := uint(claims["user_id"].(float64))
		return userID, nil
	}

	return 0, fmt.Errorf("invalid token")
}

// JWTMiddleware validates Bearer tokens and injects user ID into context.
func JWTMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				next.ServeHTTP(w, r)
				return
			}

			userID, err := ParseToken(parts[1])
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Inject user ID into context
			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext extracts user ID from request context (set by JWTMiddleware).
func GetUserIDFromContext(ctx context.Context) (uint, bool) {
	id, ok := ctx.Value(userIDContextKey).(uint)
	return id, ok
}
