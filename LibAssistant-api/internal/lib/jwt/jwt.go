package jwtValidation

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func ValidateToken(tokenString string) error {
	const op = "middleware.Jwt.ValidateToken"
	app_secret, ok := os.LookupEnv("APP_SECRET")
	if !ok {
		return fmt.Errorf("%s: %s", op, "app secret not found")
	}

	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%s: %s", op, "unexpected signing method")
		}
		return []byte(app_secret), nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}