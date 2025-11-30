package postgres

import (
	"LibAssistant_sso/internal/domain/models"
	"LibAssistant_sso/internal/storage"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Storage struct {
	db *pgx.Conn
}

// Opens connection to postgresql DB
func New(ctx context.Context, dsn string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "storage.postgres.SaveUser"

	id := time.Now().Unix()

	_, err := s.db.Exec(ctx, "INSERT INTO users(id, email, pass_hash) VALUES($1, $2, $3)", id, email, passHash)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// User returns user by email.
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.User"

	rows, err := s.db.Query(ctx, "SELECT id, email, pass_hash FROM users WHERE email = $1", email)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	var user models.User

	for rows.Next() {
		if err := rows.Scan(&user.ID, &user.Email, &user.PassHash); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
			}
			return models.User{}, fmt.Errorf("%s: %w", op, err)
		}

	}

	return user, nil
}

func (s *Storage) SaveAdmin(ctx context.Context, email string, passHash []byte) (uid int64, err error) {
	const op = "storage.postgres.SaveAdmin"

	id := time.Now().Unix()

	_, err = s.db.Exec(ctx, "INSERT INTO users(id, email, pass_hash, is_admin) VALUES($1, $2, $3, $4)", id, email, passHash, true)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
    const op = "storage.postgres.IsAdmin"

    var isAdmin bool
    err := s.db.QueryRow(ctx, "SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
        }
        return false, fmt.Errorf("%s: %w", op, err)
    }
    return isAdmin, nil
}
