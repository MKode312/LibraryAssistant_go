package main

import (
	booksgrpc "LibAssistant_api/internal/clients/books/grpc"
	ssogrpc "LibAssistant_api/internal/clients/sso/grpc"
	"LibAssistant_api/internal/config"
	isAdmin "LibAssistant_api/internal/http-server/handlers/auth/IsAdmin"
	"LibAssistant_api/internal/http-server/handlers/auth/login"
	"LibAssistant_api/internal/http-server/handlers/auth/register"
	"LibAssistant_api/internal/http-server/handlers/auth/registerAsAdmin"
	addbook "LibAssistant_api/internal/http-server/handlers/books/addBook"
	addcopies "LibAssistant_api/internal/http-server/handlers/books/addCopies"
	deletebook "LibAssistant_api/internal/http-server/handlers/books/deleteBook"
	filterbooksbygenrelist "LibAssistant_api/internal/http-server/handlers/books/filterBooksByGenreList"
	getbookbyid "LibAssistant_api/internal/http-server/handlers/books/getBookByID"
	getbookbytitle "LibAssistant_api/internal/http-server/handlers/books/getBookByTitle"
	getlistofbooks "LibAssistant_api/internal/http-server/handlers/books/getListOfBooks"
	takebook "LibAssistant_api/internal/http-server/handlers/books/takeBook"
	MWJwt "LibAssistant_api/internal/http-server/middleware/jwt"
	MWLogger "LibAssistant_api/internal/http-server/middleware/logger"
	"LibAssistant_api/internal/lib/logger/handlers/slogpretty"
	"LibAssistant_api/internal/lib/logger/sl"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	log.Info("starting api gateway")

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

	booksClient, err := booksgrpc.New(context.Background(), log, cfg.Clients.Books.Address, cfg.Clients.Books.Timeout, cfg.Clients.Books.RetriesCount)
	if err != nil {
		log.Error("failed to init books client", sl.Err(err))
		os.Exit(1)
	}

	jwtMiddleware := MWJwt.New(log)

	router.Post("/register", register.New(context.Background(), log, ssoClient))
	router.Post("/login", login.New(context.Background(), log, ssoClient))
	router.Post("/registerAsAdmin", registerAsAdmin.New(context.Background(), log, ssoClient))
	
	router.Group(func(r chi.Router) {
		r.Use(jwtMiddleware)
		r.Get("/isAdmin/{userID}", isAdmin.New(context.Background(), log, ssoClient))
		r.Post("/addBook", addbook.New(context.Background(), log, booksClient))
		r.Post("/addCopies/{bookID}", addcopies.New(context.Background(), log, booksClient))
		r.Get("/book/{bookID}", getbookbyid.New(context.Background(), log, booksClient))
		r.Get("/book/title/{title}", getbookbytitle.New(context.Background(), log, booksClient))
		r.Delete("/book/{bookID}", deletebook.New(context.Background(), log, booksClient))
		r.Put("/takeBook/{bookID}", takebook.New(context.Background(), log, booksClient))
		r.Get("/books", getlistofbooks.New(context.Background(), log, booksClient))
		r.Get("/books/{genre}", filterbooksbygenrelist.New(context.Background(), log, booksClient))
	})


	log.Info("starting http-server", slog.String("address", cfg.HTTPServer.Address))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)


	srv := &http.Server{
		Addr: cfg.HTTPServer.Address,
		Handler: router,
		ReadTimeout: cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout: cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	    }
	}()

	log.Info("http-server strarted")

	<-done
	log.Info("stopping http-server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to stop server", sl.Err(err))

		return
	}

	log.Info("http-server stopped")
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
