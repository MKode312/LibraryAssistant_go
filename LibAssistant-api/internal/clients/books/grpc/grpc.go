package booksgrpc

import (
	"LibAssistant_api/internal/domain/models"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	booksv1 "github.com/MKode312/protos/gen/go/LibAssistant/books"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	api booksv1.BooksClient
	log *slog.Logger
}

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrInternal               = errors.New("internal error")
	ErrBookExists             = errors.New("book already exists")
	ErrBookNotFound           = errors.New("book not found")
	ErrNothingToList          = errors.New("no books in the store")
	ErrNoBooksWithGenre       = errors.New("no books in the store with this genre")
	ErrNoCopiesToTake         = errors.New("no copies of this book are in the store")
	ErrNotEnoughCopiesInStore = errors.New("not enough copies of this book in the store ")
)

func New(ctx context.Context, log *slog.Logger, addr string, timeout time.Duration, retriesCount int) (*Client, error) {
	const op = "books.grpc.New"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}

	cc, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpclog.UnaryClientInterceptor(interceptorLogger(log), logOpts...),
			grpcretry.UnaryClientInterceptor(retryOpts...),
		))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Client{
		api: booksv1.NewBooksClient(cc),
		log: log,
	}, nil
}

func (c *Client) AddBook(ctx context.Context, genre string, title string, quantity int64) (string, error) {
	const op = "books.grpc.AddBook"
	resp, err := c.api.AddBook(ctx, &booksv1.AddBookRequest{
		Genre:    genre,
		Title:    title,
		Quantity: quantity,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return "", fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			}
			if st.Code() == codes.AlreadyExists {
				return "", fmt.Errorf("%s: %w", op, ErrBookExists)
			}
		}
		return "", fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetBookId(), nil
}

func (c *Client) AddCopies(ctx context.Context, bookID string, copiesToAdd int64) (bool, error) {
	const op = "books.grpc.AddCopies"
	resp, err := c.api.AddCopies(ctx, &booksv1.AddCopiesRequest{
		BookId:      bookID,
		CopiesToAdd: copiesToAdd,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return false, fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			}
			if st.Code() == codes.NotFound {
				return false, fmt.Errorf("%s: %w", op, ErrBookNotFound)
			}
		}
		return false, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetSuccess(), nil
}

func (c *Client) GetBookByID(ctx context.Context, bookID string) (models.Book, error) {
	const op = "books.grpc.GetBookByID"
	resp, err := c.api.GetBookByID(ctx, &booksv1.GetBookByIDRequest{
		BookId: bookID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return models.Book{}, fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			}
			if st.Code() == codes.NotFound {
				return models.Book{}, fmt.Errorf("%s: %w", op, ErrBookNotFound)
			}
		}
		return models.Book{}, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	book := models.Book{
		ID:              resp.GetBook().ID,
		Genre:           resp.GetBook().Genre,
		Title:           resp.GetBook().Title,
		AvailableCopies: resp.GetBook().AvailableCopies,
	}

	return book, nil
}

func (c *Client) GetBookByTitle(ctx context.Context, title string) (models.Book, error) {
	const op = "books.grpc.GetBookByTitle"
	resp, err := c.api.GetBookByTitle(ctx, &booksv1.GetBookByTitleRequest{
		Title: title,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return models.Book{}, fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			}
			if st.Code() == codes.NotFound {
				return models.Book{}, fmt.Errorf("%s: %w", op, ErrBookNotFound)
			}
		}
		return models.Book{}, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	book := models.Book{
		ID:              resp.GetBook().ID,
		Genre:           resp.GetBook().Genre,
		Title:           resp.GetBook().Title,
		AvailableCopies: resp.GetBook().AvailableCopies,
	}

	return book, nil
}

func (c *Client) TakeBook(ctx context.Context, bookID string, copiesToTake int64) (bool, error) {
	const op = "books.grpc.TakeBook"
	resp, err := c.api.TakeBook(ctx, &booksv1.TakeBookRequest{
		BookId:     bookID,
		TakeCopies: copiesToTake,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return false, fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			case codes.NotFound:
				return false, fmt.Errorf("%s: %w", op, ErrBookNotFound)
			case codes.OutOfRange:
				return false, fmt.Errorf("%s: %w", op, ErrNoCopiesToTake)
			case codes.FailedPrecondition:
				return false, fmt.Errorf("%s: %w", op, ErrNotEnoughCopiesInStore)
			}
		}
		return false, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetSuccess(), nil
}

func (c *Client) DeleteBook(ctx context.Context, bookID string) (bool, error) {
	const op = "books.grpc.DeleteBook"
	resp, err := c.api.DeleteBook(ctx, &booksv1.DeleteBookRequest{
		BookId: bookID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return false, fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			}
			if st.Code() == codes.NotFound {
				return false, fmt.Errorf("%s: %w", op, ErrBookNotFound)
			}
		}
		return false, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetSuccess(), nil
}

func (c *Client) GetListOfBooks(ctx context.Context) ([]models.Book, error) {
	const op = "books.grpc.GetListOfBooks"
	resp, err := c.api.GetListOfBooks(ctx, &booksv1.GetListOfBooksRequest{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return nil, fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			}
			if st.Code() == codes.OutOfRange {
				return nil, fmt.Errorf("%s: %w", op, ErrNothingToList)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	var books []models.Book

	for _, book := range resp.GetBooks() {
		modelBook := models.Book{
			ID:              book.ID,
			Genre:           book.Genre,
			Title:           book.Title,
			AvailableCopies: book.AvailableCopies,
		}
		books = append(books, modelBook)
	}

	return books, nil
}

func (c *Client) FilterBooksByGenreList(ctx context.Context, genre string) ([]models.Book, error) {
	const op = "books.grpc.FilterBooksByGenreList"
	resp, err := c.api.GetListOfBooks(ctx, &booksv1.GetListOfBooksRequest{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return nil, fmt.Errorf("%s: %w", op, ErrInvalidRequest)
			}
			if st.Code() == codes.NotFound {
				return nil, fmt.Errorf("%s: %w", op, ErrNoBooksWithGenre)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	var books []models.Book

	for _, book := range resp.GetBooks() {
		modelBook := models.Book{
			ID:              book.ID,
			Genre:           book.Genre,
			Title:           book.Title,
			AvailableCopies: book.AvailableCopies,
		}
		books = append(books, modelBook)
	}

	return books, nil
}

func interceptorLogger(l *slog.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, level grpclog.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(level), msg, fields...)
	})
}
