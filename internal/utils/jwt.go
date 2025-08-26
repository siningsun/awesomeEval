package utils

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

var (
	SecretKey      = "awesome-secret-key-for-jwt"
	ExpireDuration = time.Hour * 24
)

// GenerateJWT generates a JWT token with the given user ID and secret key.
func GenerateJWT(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(ExpireDuration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

// ParseJWT parses the JWT token and returns the user ID if valid.
func ParseJWT(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(SecretKey), nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userID, ok := claims["user_id"].(float64); ok {
			return int64(userID), nil
		}
	}
	return 0, errors.New("no user_id found in token claims")
}
