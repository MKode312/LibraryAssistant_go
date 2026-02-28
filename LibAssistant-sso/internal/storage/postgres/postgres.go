package postgres

import (
	"LibAssistant_sso/internal/domain/models"
	"LibAssistant_sso/internal/storage"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (userID string, err error) {
	const op = "storage.postgres.SaveUser"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	id := uuid.New().String()

	_, err = tx.Exec(ctx, "INSERT INTO users(id, email, pass_hash) VALUES($1, $2, $3)", id, email, passHash)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return "", fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// User returns user by email.
func (s *Storage) User(ctx context.Context, email string) (user models.User, err error) {
	const op = "storage.postgres.User"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	rows, err := tx.Query(ctx, "SELECT id, email, pass_hash FROM users WHERE email = $1", email)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

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

func (s *Storage) DeleteUserByID(ctx context.Context, userID string) (success bool, err error) {
	const op = "storage.postgres.DeleteUserByID"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	if !exists {
		return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}

	_, err = tx.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (s *Storage) SaveAdmin(ctx context.Context, email string, passHash []byte) (uid string, err error) {
	const op = "storage.postgres.SaveAdmin"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	id := uuid.New().String()

	_, err = tx.Exec(ctx, "INSERT INTO users(id, email, pass_hash, is_admin) VALUES($1, $2, $3, $4)", id, email, passHash, true)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return "", fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID string) (isAdmin bool, err error) {
	const op = "storage.postgres.IsAdmin"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, "SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return isAdmin, nil
}
