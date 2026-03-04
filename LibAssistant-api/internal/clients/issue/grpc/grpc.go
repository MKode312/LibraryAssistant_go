package issuegrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	issuev1 "github.com/MKode312/protos/gen/go/LibAssistant/issue"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	log *slog.Logger
	api issuev1.IssueServiceClient
}

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInternal           = errors.New("internal error")
	ErrAlreadyActiveIssue = errors.New("this student already has an active issue for this book")
)

func New(ctx context.Context, log *slog.Logger, addr string, timeout time.Duration, retriesCount int) (*Client, error) {
	const op = "issue.grpc.New"
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
		api: issuev1.NewIssueServiceClient(cc),
		log: log,
	}, nil

}

func (c *Client) IssueBook(ctx context.Context, daysDue int32, studentID string, bookID string) (string, error) {
	const op = "issue.grpc.IssueBook"
	resp, err := c.api.IssueBook(ctx, &issuev1.IssueRequest{
		StudentId: studentID,
		BookId:    bookID,
		DaysDue:   daysDue,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return "", fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.FailedPrecondition {
				return "", fmt.Errorf("%s: %w", op, ErrAlreadyActiveIssue)
			}
			if st.Code() == codes.Internal {
				return "", fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return "", fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetIssueId(), nil
}

func interceptorLogger(l *slog.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, level grpclog.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(level), msg, fields...)
	})
}
