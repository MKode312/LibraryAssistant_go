package booksgrpc

import (
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
	ErrInvalidRequest = errors.New("invalid request")
	ErrNotFound       = errors.New("not found")
	ErrInternal       = errors.New("internal error")
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

func (c *Client) AddBook(ctx context.Context, genre, title string, quantity int64) (string, error) {
	resp, err := c.api.AddBook(ctx, &booksv1.AddBookRequest{
		Genre:    genre,
		Title:    title,
		Quantity: quantity,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return "", ErrInvalidRequest
			}
		}
		return "", ErrInternal
	}

	return resp.GetBookID(), nil
}

func (c *Client) ListBooks(ctx context.Context) ([]*booksv1.Book, error) {
	resp, err := c.api.GetListOfBooks(ctx, &booksv1.GetListOfBooksRequest{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.NotFound {
				return nil, ErrNotFound
			}
		}
		return nil, ErrInternal
	}

	return resp.GetBooks(), nil
}

func interceptorLogger(l *slog.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, level grpclog.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(level), msg, fields...)
	})
}
