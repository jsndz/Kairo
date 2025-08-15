package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)


func GenerateJWT(email string,userId uint32) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal().Msg("Secret is Empty")
	}
	claims := jwt.MapClaims{
		"id"  : userId,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}


func GenerateJWTForWS(docId uint32,userId uint32) (string, error) {
	jwtSecret := os.Getenv("WS_JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal().Msg("Secret is Empty")
	}
	claims := jwt.MapClaims{
		"id"  : userId,
		"doc_id":docId,
		"type":"ws",
		"exp":     time.Now().Add(time.Minute * 5).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}