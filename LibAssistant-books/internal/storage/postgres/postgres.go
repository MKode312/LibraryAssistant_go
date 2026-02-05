package postgres

import (
	"LibAssistant_books/internal/domain/models"
	"LibAssistant_books/internal/storage"
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

func (s *Storage) AddBook(ctx context.Context, genre string, title string, quantity int64) (string, error) {
	const op = "storage.postgres.AddBook"

	id := uuid.New().String()

	_, err := s.db.Exec(ctx, "INSERT INTO books(bookID, genre, title, quantity) VALUES($1, $2, $3, $4)", id, genre, title, quantity)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return "", fmt.Errorf("%s: %w", op, storage.ErrBookExists)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetBook(ctx context.Context, bookID string) (models.Book, error) {
	const op = "storage.postgres.GetBook"

	rows, err := s.db.Query(ctx, "SELECT bookID, genre, title, quantity FROM books WHERE bookID = $1", bookID)
	if err != nil {
		return models.Book{}, fmt.Errorf("%s: %w", op, err)
	}

	var book models.Book

	for rows.Next() {
		if err := rows.Scan(&book.ID, &book.Genre, &book.Title, &book.AvailableCopies); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return models.Book{}, fmt.Errorf("%s: %w", op, storage.ErrBookNotFound)
			}
			return models.Book{}, fmt.Errorf("%s: %w", op, err)
		}

	}

	return book, nil
}

func (s *Storage) TakeBook(ctx context.Context, bookID string, copiesToTake int64) (bool, error) {
	const op = "storage.postgres.TakeBook"

	rows, err := s.db.Query(ctx, "SELECT quantity FROM books WHERE bookID = $1", bookID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	var copiesInStore int64

	for rows.Next() {
		if err := rows.Scan(&copiesInStore); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, fmt.Errorf("%s: %w", op, storage.ErrBookNotFound)
			}
			return false, fmt.Errorf("%s: %w", op, err)
		}
	}

	if copiesInStore <= 0 {
		return false, fmt.Errorf("%s: %w", op, storage.ErrNoCopiesToTake)
	}

	if copiesInStore < copiesToTake {
		return false, fmt.Errorf("%s: %w", op, storage.ErrNotEnoughCopiesInStore)
	}

	newCopiesInStore := copiesInStore - copiesToTake

	_, err = s.db.Exec(ctx, "UPDATE books SET quantity = $1 WHERE bookID = $2", newCopiesInStore, bookID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (s *Storage) DeleteBook(ctx context.Context, bookID string) (bool, error) {
	const op = "storage.postgres.DeleteBook"

	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM books WHERE bookID = $1)", bookID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	if !exists {
		return false, fmt.Errorf("%s: %w", op, storage.ErrBookNotFound)
	}

	_, err = s.db.Exec(ctx, "DELETE FROM books WHERE bookID = $1", bookID)
	if err != nil {
		return false, fmt.Errorf("%s: failed to execute delete: %w", op, err)
	}

	return true, nil
}

func (s *Storage) GetListOfBooks(ctx context.Context) ([]models.Book, error) {
	const op = "storage.postgres.GetListOfBooks"

	rows, err := s.db.Query(ctx, "SELECT bookID, genre, title, quantity FROM books")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var books []models.Book

	for rows.Next() {
		var book models.Book
		if err := rows.Scan(&book.ID, &book.Genre, &book.Title, &book.AvailableCopies); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%s: %w", op, storage.ErrNothingToList)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		books = append(books, book)
	}

	return books, nil
}

func (s *Storage) FilterBooksByGenreList(ctx context.Context, genre string) ([]models.Book, error) {
	const op = "storage.postgres.FilterbooksByGenreList"

	rows, err := s.db.Query(ctx, "SELECT bookID, genre, title, quantity FROM books WHERE genre = $1", genre)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var books []models.Book

	for rows.Next() {
		var book models.Book
		if err := rows.Scan(&book.ID, &book.Genre, &book.Title, &book.AvailableCopies); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%s: %w", op, storage.ErrNoBooksWithGenre)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		books = append(books, book)
	}

	return books, nil
}
