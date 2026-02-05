package app

import (
	grpcapp "LibAssistant_books/internal/app/grpc"
	"LibAssistant_books/internal/services/books"
	"LibAssistant_books/internal/storage/postgres"
	"context"
	"log/slog"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(ctx context.Context, log *slog.Logger, dsn string, grpcPort int) *App {
	storage, err := postgres.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	booksService := books.New(log, storage, storage, storage)

	grpcApp := grpcapp.New(log, booksService, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}
