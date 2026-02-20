package addcopies

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

type Request struct {
	CopiesToAdd int64  `json:"copiesToAdd" validate:"required"`
}

type Response struct {
	resp.Response
	Success bool `json:"success"`
}

func New(ctx context.Context, log *slog.Logger, booksClient *booksgrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Books.AddCopies.New"

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
		copiesToAdd := req.CopiesToAdd

		success, err := booksClient.AddCopies(ctx, bookID, copiesToAdd)
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

			log.Error("failed to add copies", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		if success {
			log.Info("copies added successfully")
			w.WriteHeader(http.StatusAccepted)
			render.JSON(w, r, Response{
				Response: resp.OK(),
				Success: success,
			})
		} else {
			log.Warn("copies were not added!")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, Response{
				Response: resp.OK(),
				Success: success,
			})
		}
	}
}
