package app

import (
	"LibAssistant_sso/internal/app/grpc"
	"LibAssistant_sso/internal/services/auth"
	"LibAssistant_sso/internal/storage/postgres"
	"context"
	"log/slog"
	"time"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(ctx context.Context, log *slog.Logger, dsn string, grpcPort int, tokenTTL time.Duration) *App {
	storage, err := postgres.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage, tokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}