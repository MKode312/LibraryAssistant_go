package deletebook

import (
	booksgrpc "LibAssistant_api/internal/clients/books/grpc"
	resp "LibAssistant_api/internal/lib/api/response"
	"LibAssistant_api/internal/lib/logger/sl"
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {}

type Response struct {
	resp.Response
	Success bool `json:"success"`
}

func New(ctx context.Context, log *slog.Logger, booksClient *booksgrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Books.DeleteBook.New"

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

		bookID := chi.URLParam(r, "bookID")

		success, err := booksClient.DeleteBook(ctx, bookID)
		if err != nil {
			if errors.Is(err, booksgrpc.ErrInvalidRequest) {
				log.Error("invalid request")
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid request"))
				return
			}

			if errors.Is(err, booksgrpc.ErrBookNotFound) {
				log.Error("book not found")
				w.WriteHeader(http.StatusUnprocessableEntity)
				render.JSON(w, r, resp.Error("Book not found"))
				return
			}

			if errors.Is(err, booksgrpc.ErrInternal) {
				log.Error("internal error", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Unknown error"))
				return
			}

			log.Error("failed to delete book", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		if success {
			log.Info("book was deleted successfully")
			w.WriteHeader(http.StatusOK)
			render.JSON(w, r, Response{
				Response: resp.OK(),
				Success: success,
			})
		} else {
			log.Warn("book was not deleted!")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, Response{
				Response: resp.OK(),
				Success: success,
			})
		}
	}
}
