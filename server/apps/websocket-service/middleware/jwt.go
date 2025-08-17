package middleware

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func Authenticate(token string) (string, error)  {
	wsjwtSecret := os.Getenv("WS_JWT_SECRET")
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(wsjwtSecret), nil
	})
	log.Println(err)
	if err != nil || !parsedToken.Valid {
		return "",  fmt.Errorf("invalid token")
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims);
	if !ok {
   }
   var userID string

   switch id := claims["id"].(type) {
	case string:
		userID = id
	case float64:
		userID = fmt.Sprintf("%.0f", id) 
	default:
		return "", fmt.Errorf("user ID is missing or invalid")
	}
   return userID, nil

}