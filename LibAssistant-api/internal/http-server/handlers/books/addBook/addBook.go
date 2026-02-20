package addbook

import (
	booksgrpc "LibAssistant_api/internal/clients/books/grpc"
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

type Request struct {
	Genre string `json:"genre" validate:"required"`
	Title string `json:"title" validate:"required"`
	Quantity int64 `json:"quantity" validate:"required"`
}

type Response struct {
	resp.Response
	BookID string `json:"bookID"`
}

func New(ctx context.Context, log *slog.Logger, booksClient *booksgrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Book.AddBook.New"

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

		genre := req.Genre
		title := req.Title
		quantity := req.Quantity

		bookID, err := booksClient.AddBook(ctx, genre, title, quantity)
		if err != nil {
			if errors.Is(err, booksgrpc.ErrInvalidRequest) {
				log.Error("invalid request")

				w.WriteHeader(http.StatusBadRequest)

				render.JSON(w, r, resp.Error("Invalid request"))

				return 
			}

			if errors.Is(err, booksgrpc.ErrBookExists) {
				log.Error("book already exists")
				
				w.WriteHeader(http.StatusConflict)

				render.JSON(w, r, resp.Error("You cannot add the existing book"))

				return
			}

			if errors.Is(err, booksgrpc.ErrInternal) {
				log.Error("internal error", sl.Err(err))

				w.WriteHeader(http.StatusInternalServerError)

				render.JSON(w, r, resp.Error("Unknown error"))

				return
			}

			log.Error("failed to add new book", sl.Err(err))

			w.WriteHeader(http.StatusInternalServerError)

			render.JSON(w, r, resp.Error("Unknown error"))

			return
		}
		
		log.Info("book added", slog.String("id", bookID))

		w.WriteHeader(http.StatusCreated)

		render.JSON(w, r, Response{
			Response: resp.OK(),
			BookID: bookID,
		})
	}
}