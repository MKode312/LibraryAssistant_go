package jwt

import (
	"LibAssistant_sso/internal/domain/models"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	_ "github.com/joho/godotenv/autoload"
)

func NewToken(user models.User, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(duration).Unix()

	app_secret, ok := os.LookupEnv("APP_SECRET")
	if !ok {
		return "", errors.New("app_secret not found")
	}

	tokenString, err := token.SignedString([]byte(app_secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
