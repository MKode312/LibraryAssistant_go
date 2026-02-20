package books

import (
	"LibAssistant_books/internal/domain/models"
	"LibAssistant_books/internal/lib/logger/sl"
	"LibAssistant_books/internal/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type Books struct {
	log          *slog.Logger
	bookSaver    BookSaver
	bookProvider BookProvider
	bookLister   BookLister
}

type BookSaver interface {
	AddBook(ctx context.Context, genre string, title string, quantity int64) (string, error)
	AddCopies(ctx context.Context, bookID string, copiesToAdd int64) (bool, error)
}
type BookProvider interface {
	TakeBook(ctx context.Context, bookID string, take_copies int64) (bool, error)
	GetBookByID(ctx context.Context, bookID string) (models.Book, error)
	GetBookByTitle(ctx context.Context, title string) (models.Book, error)
	DeleteBook(ctx context.Context, bookID string) (bool, error)
}

type BookLister interface {
	GetListOfBooks(ctx context.Context) ([]models.Book, error)
	FilterBooksByGenreList(ctx context.Context, genre string) ([]models.Book, error)
}

func New(log *slog.Logger, bookSaver BookSaver, bookProvider BookProvider, bookLister BookLister) *Books {
	return &Books{
		log:          log,
		bookSaver:    bookSaver,
		bookProvider: bookProvider,
		bookLister:   bookLister,
	}
}

var (
	ErrBookExists = errors.New("book already exists")
	ErrBookNotFound = errors.New("book not found")
	ErrNothingToList = errors.New("no books in the store")
	ErrNoBooksWithGenre = errors.New("no books in the store with this genre")
	ErrNoCopiesToTake = errors.New("no copies of this book are in the store")
	ErrNotEnoughCopiesInStore = errors.New("not enough copies of this book in the store ")
)

func (b *Books) AddBook(ctx context.Context, genre string, title string, quantity int64) (string, error) {
	const op = "books.AddBook"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("adding a book")

	id, err := b.bookSaver.AddBook(ctx, genre, title, quantity)
	if err != nil {
		if errors.Is(err, storage.ErrBookExists) {
			log.Error("book not found", sl.Err(err))

			return "", fmt.Errorf("%s: %w", op, ErrBookExists)
		}
		log.Error("failed to add a book", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("book saved")

	return id, nil
}

func (b *Books) AddCopies(ctx context.Context, bookID string, copiesToAdd int64) (bool, error) {
	const op = "books.AddCopies"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("adding copies")

	success, err :=b.bookSaver.AddCopies(ctx, bookID, copiesToAdd)
	if err != nil {
		if errors.Is(err, storage.ErrBookNotFound) {
			log.Error("book not found", sl.Err(err))

			return false, fmt.Errorf("%s: %w", op, ErrBookNotFound)
		}
		log.Error("failed to add copies", sl.Err(err))

		return false, fmt.Errorf("%s: %w", op, err)
	}

	if success {
		log.Info("copies added")
	} else {
		log.Warn("copies were not added!")
	}

	return success, nil
}

func (b *Books) TakeBook(ctx context.Context, bookID string, take_copies int64) (bool, error) {
	const op = "books.TakeBook"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("taking a book")

	success, err := b.bookProvider.TakeBook(ctx, bookID, take_copies)
	if err != nil {
		if errors.Is(err, storage.ErrBookNotFound) {
			log.Error("book not found", sl.Err(err))

			return false, fmt.Errorf("%s: %w", op, ErrBookNotFound)
		}
		
		if errors.Is(err, storage.ErrNoCopiesToTake) {
			log.Error("no copies to take", sl.Err(err))

			return false, fmt.Errorf("%s: %w", op, ErrNoCopiesToTake)
		}

		if errors.Is(err, ErrNotEnoughCopiesInStore) {
			log.Error("not enough copies in the store", sl.Err(err))

			return false, fmt.Errorf("%s: %w", op, ErrNotEnoughCopiesInStore)
		}
		log.Error("failed to take a book", sl.Err(err))

		return false, fmt.Errorf("%s: %w", op, err)
	}

	if success {
		log.Info("book was successfully taken")
	} else {
		log.Warn("book was not taken!")
	}

	return success, nil
}

func (b *Books) GetBookByID(ctx context.Context, bookID string) (models.Book, error) {
	const op = "books.GetBook"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("getting a book")

	book, err := b.bookProvider.GetBookByID(ctx, bookID)
	if err != nil {
		if errors.Is(err, storage.ErrBookNotFound) {
			log.Error("book not found", sl.Err(err))

			return models.Book{}, fmt.Errorf("%s: %w", op, ErrBookNotFound)
		}
		log.Error("failed to get a book", sl.Err(err))

		return models.Book{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("book was received")

	return book, nil
}

func (b *Books) GetBookByTitle(ctx context.Context, title string) (models.Book, error) {
	const op = "books.GetBookByTitle"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("getting a book")

	book, err := b.bookProvider.GetBookByTitle(ctx, title)
	if err != nil {
		if errors.Is(err, storage.ErrBookNotFound) {
			log.Error("book not found", sl.Err(err))

			return models.Book{}, fmt.Errorf("%s: %w", op, ErrBookNotFound)
		}
		log.Error("failed to get a book", sl.Err(err))

		return models.Book{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Error("book was received")

	return book, nil
}

func (b *Books) DeleteBook(ctx context.Context, bookID string) (bool, error) {
	const op = "books.DeleteBook"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("deleting a book")

	success, err := b.bookProvider.DeleteBook(ctx, bookID)
	if err != nil {
		if errors.Is(err, storage.ErrBookNotFound) {
			log.Error("book not found", sl.Err(err))

			return false, fmt.Errorf("%s: %w", op, ErrBookNotFound)
		}
		log.Error("failed to delete a book", sl.Err(err))

		return false, fmt.Errorf("%s: %w", op, err)
	}

	if success {
		log.Info("book was successfully deleted")
	} else {
		log.Warn("book was not deleted!")
	}

	return success, nil
}

func (b *Books) GetListOfBooks(ctx context.Context) ([]models.Book, error) {
	const op = "books.GetListOfBooks"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("getting list of books")

	books, err := b.bookLister.GetListOfBooks(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNothingToList) {
			log.Error("no books to list")

			return nil, fmt.Errorf("%s: %w", op, ErrNothingToList)
		}
		log.Error("failed to get a list of books", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("list of books was got")

	return books, nil
}

func (b *Books) FilterBooksByGenreList(ctx context.Context, genre string) ([]models.Book, error) {
	const op = "books.TakeBook"

	log := b.log.With(
		slog.String("op", op),
	)

	log.Info("getting filtered by genre list of books")

	books, err := b.bookLister.FilterBooksByGenreList(ctx, genre)
	if err != nil {
		if errors.Is(err, storage.ErrNoBooksWithGenre) {
			log.Error("no books with this genre")

			return nil, fmt.Errorf("%s: %w", op, ErrNoBooksWithGenre)
		}
		log.Error("failed to get filtered by genre list of books", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("filtered list was got")

	return books, nil
}
