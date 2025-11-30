package auth

import (
	"LibAssistant_sso/internal/domain/models"
	"LibAssistant_sso/internal/lib/jwt"
	"LibAssistant_sso/internal/lib/logger/sl"
	"LibAssistant_sso/internal/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	log          *slog.Logger
	userSaver    UserSaver
	userProvider UserProvider
	tokenTTL     time.Duration
}

type UserSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte) (uid int64, err error)
	SaveAdmin(ctx context.Context, email string, passHash []byte) (uid int64, err error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWrongAdminSecret = errors.New("the provided admin secret key is wrong")
	ErrUserExists = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

func New(log *slog.Logger, userSaver UserSaver, userProvider UserProvider, tokenTTL time.Duration) *Auth {
	return &Auth{
		userSaver:    userSaver,
		userProvider: userProvider,
		log:          log,
		tokenTTL:     tokenTTL,
	}
}

func (a *Auth) Login(ctx context.Context, email string, password string) (string, error) {
	const op = "auth.Login"

	log := a.log.With(slog.String("op", op))

	log.Info("attempting to login user")

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", sl.Err(err))

			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		log.Error("failed to get user", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Info("invalid credentials", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	log.Info("user logged in successfully")

	token, err := jwt.NewToken(user, a.tokenTTL)
	if err != nil {
		log.Error("failed to generate token", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (a *Auth) RegisterNewUser(ctx context.Context, email string, password string) (int64, error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(slog.String("op", op))

	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.userSaver.SaveUser(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Error("user already exists", sl.Err(err))

			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered")

	return id, nil
}

func (a *Auth) RegisterNewAdmin(ctx context.Context, email string, password string, admin_secret string) (int64, error) {
	const op = "auth.RegisterNewAdmin"

	log := a.log.With(slog.String("op", op))

	log.Info("registering admin")

	adminPswd, ok := os.LookupEnv("ADMIN_SECRET")
	if !ok {
		log.Error("failed to find admin_secret")

		return 0, fmt.Errorf("%s: %w", op, errors.New("admin_key not found"))
	}

	if admin_secret != adminPswd {
		log.Error("wrong admin secret key")

		return 0, fmt.Errorf("%s: %w", op, ErrWrongAdminSecret)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.userSaver.SaveAdmin(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Error("user already exists", sl.Err(err))
			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save admin", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("admin registered")

	return id, nil
}

func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "auth.IsAdmin"

	log := a.log.With(slog.String("op", op))

	log.Info("checking if user is admin")

	isAdmin, err := a.userProvider.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Error("user not found", sl.Err(err))
			return false, fmt.Errorf("%s: %w", op, ErrUserNotFound)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked if user is admin")

	return isAdmin, nil
}
