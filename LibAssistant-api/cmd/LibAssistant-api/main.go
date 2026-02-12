package main

import (
	ssogrpc "LibAssistant_api/internal/clients/sso/grpc"
	"LibAssistant_api/internal/config"
	isAdmin "LibAssistant_api/internal/http-server/handlers/auth/IsAdmin"
	"LibAssistant_api/internal/http-server/handlers/auth/login"
	"LibAssistant_api/internal/http-server/handlers/auth/register"
	"LibAssistant_api/internal/http-server/handlers/auth/registerAsAdmin"
	MWLogger "LibAssistant_api/internal/http-server/middleware/logger"
	"LibAssistant_api/internal/lib/logger/handlers/slogpretty"
	"LibAssistant_api/internal/lib/logger/sl"
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envLocal = "local"
	envProd  = "prod"
	envDev   = "dev"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("starting application")

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(MWLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	ssoClient, err := ssogrpc.New(context.Background(), log, cfg.Clients.SSO.Address, cfg.Clients.SSO.Timeout, cfg.Clients.SSO.RetriesCount)
	if err != nil {
		log.Error("failed to init sso client", sl.Err(err))
		os.Exit(1)
	}

	router.Post("/register", register.New(context.Background(), log, ssoClient))
	router.Post("/login", login.New(context.Background(), log, ssoClient))
	router.Post("/isAdmin", isAdmin.New(context.Background(), log, ssoClient))
	router.Post("/registerAsAdmin", registerAsAdmin.New(context.Background(), log, ssoClient))

	srv := &http.Server{
		Addr: cfg.HTTPServer.Address,
		Handler: router,
		ReadTimeout: cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout: cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error("failed to stop server")
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
