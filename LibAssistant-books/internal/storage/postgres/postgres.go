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

// Adds a new book to storage
func (s *Storage) AddBook(ctx context.Context, genre string, title string, quantity int64) (id string, err error) {
	const op = "storage.postgres.AddBook"

	id = uuid.New().String()

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

	_, err = tx.Exec(ctx, "INSERT INTO books(bookID, genre, title, quantity) VALUES($1, $2, $3, $4)", id, genre, title, quantity)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return "", fmt.Errorf("%s: %w", op, storage.ErrBookExists)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// Adds some copies to the existing book
func (s *Storage) AddCopies(ctx context.Context, bookID string, copiesToAdd int64) (success bool, err error) {
	const op = "storage.postgres.AddCopies"

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

	rows, err := tx.Query(ctx, "SELECT quantity FROM books WHERE bookID = $1", bookID)
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

	newCopiesInStore := copiesInStore + copiesToAdd

	_, err = tx.Exec(ctx, "UPDATE books SET quantity = $1 WHERE bookID = $2", newCopiesInStore, bookID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

// Gets the book by it's ID from storage (uuid)
func (s *Storage) GetBookByID(ctx context.Context, bookID string) (book models.Book, err error) {
	const op = "storage.postgres.GetBookByID"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Book{}, fmt.Errorf("%s: %w", op, err)
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

	err = tx.QueryRow(ctx, "SELECT bookID, genre, title, quantity FROM books WHERE bookID = $1", bookID).Scan(&book.ID, &book.Genre, &book.Title, &book.AvailableCopies)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Book{}, fmt.Errorf("%s: %w", op, storage.ErrBookNotFound)
		}
		return models.Book{}, fmt.Errorf("%s: %w", op, err)
	}

	
	return book, nil
}

// Gets the book by it's title from the storage
func (s *Storage) GetBookByTitle(ctx context.Context, title string) (book models.Book, err error) {
	const op = "storage.postgres.GetBookByTitle"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Book{}, fmt.Errorf("%s: %w", op, err)
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

	err = tx.QueryRow(ctx, "SELECT bookID, genre, title, quantity FROM books WHERE title = $1", title).Scan(&book.ID, &book.Genre, &book.Title, &book.AvailableCopies)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Book{}, fmt.Errorf("%s: %w", op, storage.ErrBookNotFound)
		}
		return models.Book{}, fmt.Errorf("%s: %w", op, err)
	}

	return book, nil
}

// Takes some copies of the book by it's ID (uuid) from the storage
func (s *Storage) TakeBook(ctx context.Context, bookID string, copiesToTake int64) (bool, error) {
	const op = "storage.postgres.TakeBook"

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

	rows, err := tx.Query(ctx, "SELECT quantity FROM books WHERE bookID = $1", bookID)
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

	_, err = tx.Exec(ctx, "UPDATE books SET quantity = $1 WHERE bookID = $2", newCopiesInStore, bookID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

// Deletes book from the db by it's ID (uuid)
func (s *Storage) DeleteBook(ctx context.Context, bookID string) (bool, error) {
	const op = "storage.postgres.DeleteBook"

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
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM books WHERE bookID = $1)", bookID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	if !exists {
		return false, fmt.Errorf("%s: %w", op, storage.ErrBookNotFound)
	}

	_, err = tx.Exec(ctx, "DELETE FROM books WHERE bookID = $1", bookID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

// Gets all books that are in the storage
func (s *Storage) GetListOfBooks(ctx context.Context) (books []models.Book, err error) {
	const op = "storage.postgres.GetListOfBooks"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
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

	rows, err := tx.Query(ctx, "SELECT bookID, genre, title, quantity FROM books")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

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

// Gets only books with the specified genre from the storage
func (s *Storage) FilterBooksByGenreList(ctx context.Context, genre string) ([]models.Book, error) {
	const op = "storage.postgres.FilterbooksByGenreList"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
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

	rows, err := tx.Query(ctx, "SELECT bookID, genre, title, quantity FROM books WHERE genre = $1", genre)
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
