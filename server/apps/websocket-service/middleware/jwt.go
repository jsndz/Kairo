package middleware

import (
	"fmt"
	"os"
	"strconv"

	"github.com/golang-jwt/jwt"
)

func Authenticate(token string) (uint32, error) {
	secret := os.Getenv("WS_JWT_SECRET")
	if secret == "" {
		return 0, fmt.Errorf("missing WS_JWT_SECRET")
	}

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !parsedToken.Valid {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid claims")
	}

	var userID string
	switch id := claims["id"].(type) {
	case string:
		userID = id
	case float64:
		userID = fmt.Sprintf("%.0f", id)
	default:
		return 0, fmt.Errorf("user ID missing or invalid")
	}

	parsedUserId, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID")
	}
	return uint32(parsedUserId), nil
}
