package getlistofbooks

import (
	booksgrpc "LibAssistant_api/internal/clients/books/grpc"
	"LibAssistant_api/internal/domain/models"
	resp "LibAssistant_api/internal/lib/api/response"
	"LibAssistant_api/internal/lib/logger/sl"
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct{}

type Response struct {
	resp.Response
	Books []models.Book `json:"books"`
}

func New(ctx context.Context, log *slog.Logger, booksClient *booksgrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Books.GetListOfBooks.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validationErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request"))
			render.JSON(w, r, resp.ValidationError(validationErr))
			return
		}

		books, err := booksClient.GetListOfBooks(ctx)
		if err != nil {
			if errors.Is(err, booksgrpc.ErrInvalidRequest) {
				log.Error("invalid request")
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid request"))
				return
			}

			if errors.Is(err, booksgrpc.ErrNothingToList) {
				log.Error("nothing to list")
				w.WriteHeader(http.StatusNoContent)
				render.JSON(w, r, resp.Error("Nothing to list"))
				return
			}

			if errors.Is(err, booksgrpc.ErrInternal) {
				log.Error("internal error", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Unknown error"))
				return
			}

			log.Error("failed to get list of books", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		log.Info("list of books was received")
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Books: books,
		})
	}
}
