package ssogrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	ssov1 "github.com/MKode312/protos/gen/go/LibAssistant/sso"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	api ssov1.AuthClient
	log *slog.Logger
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInternal           = errors.New("internal error")
	ErrWrongAdminSecret   = errors.New("the provided admin secret key is wrong")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
)

func New(ctx context.Context, log *slog.Logger, addr string, timeout time.Duration, retriesCount int) (*Client, error) {
	const op = "grpc.New"

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
		api: ssov1.NewAuthClient(cc),
		log: log,
	}, nil
}

func (c *Client) RegisterNewUser(ctx context.Context, email string, password string) (string, error) {
	const op = "sso.grpc.RegisterNewUser"

	resp, err := c.api.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return "", fmt.Errorf("%s: %w: %s", op, ErrInvalidCredentials, st.Message())
			}

			if st.Code() == codes.AlreadyExists {
				return "", fmt.Errorf("%s: %w", op, ErrUserExists)
			}
		}
		return "", fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetUserId(), nil
}

func (c *Client) RegisterNewAdmin(ctx context.Context, email, passowrd, admin_secret string) (string, error) {
	const op = "sso.grpc.RegisterNewAdmin"

	resp, err := c.api.RegisterAsAdmin(ctx, &ssov1.RegisterAsAdminRequest{
		Email:       email,
		Password:    passowrd,
		AdminSecret: admin_secret,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.PermissionDenied {
				return "", fmt.Errorf("%s: %w", op, ErrWrongAdminSecret)
			}

			if st.Code() == codes.AlreadyExists {
				return "", fmt.Errorf("%s: %w", op, ErrUserExists)
			}

			if st.Code() == codes.InvalidArgument {
				return "", fmt.Errorf("%s: %w: %s", op, ErrInvalidCredentials, st.Message())
			}
		}
		return "", fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetUserId(), nil
}

func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	const op = "sso.grpc.Login"

	resp, err := c.api.Login(ctx, &ssov1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return "", fmt.Errorf("%s: %w: %s", op, ErrInvalidCredentials, st.Message())
			}

			return "", fmt.Errorf("%s: %w", op, ErrInternal)
		}
	}

	return resp.GetToken(), nil
}

func (c *Client) DeleteUserByID(ctx context.Context, userID string) (bool, error) {
	const op = "sso.grpc.DeleteUserByID"

	resp, err := c.api.DeleteUserByID(ctx, &ssov1.DeleteUserByIDRequest{
		UserId: userID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return false, fmt.Errorf("%s: %w: %s", op, ErrInvalidCredentials, st.Message())
			}
			if st.Code() == codes.NotFound {
				return false, fmt.Errorf("%s: %w", op, ErrUserNotFound)
			}
		}
		return false, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetSuccess(), nil
}
func (c *Client) IsAdmin(ctx context.Context, userID string) (bool, error) {
	const op = "sso.grpc.IsAdmin"

	resp, err := c.api.IsAdmin(ctx, &ssov1.IsAdminRequest{
		UserId: userID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return false, fmt.Errorf("%s: %w: %s", op, ErrInvalidCredentials, st.Message())
			}
			if st.Code() == codes.NotFound {
				return false, fmt.Errorf("%s: %w", op, ErrUserNotFound)
			}
		}
		return false, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetIsAdmin(), nil
}

func interceptorLogger(l *slog.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, level grpclog.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(level), msg, fields...)
	})
}
