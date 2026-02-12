package list

import (
	booksgrpc "LibAssistant_api/internal/clients/books/grpc"
	"LibAssistant_api/internal/lib/api/response"
	"LibAssistant_api/internal/lib/logger/sl"
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Book struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Author          string `json:"author"`
	ISBN            string `json:"isbn"`
	Category        string `json:"category"`
	TotalCopies     int64  `json:"totalCopies"`
	AvailableCopies int64  `json:"availableCopies"`
	Location        string `json:"location"`
}

type Response struct {
	resp.Response
	Books []Book `json:"books"`
}

func New(ctx context.Context, log *slog.Logger, booksClient *booksgrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Books.List.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		books, err := booksClient.ListBooks(ctx)
		if err != nil {
			log.Error("failed to list books", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		result := make([]Book, 0, len(books))
		for _, b := range books {
			result = append(result, Book{
				ID:              b.GetID(),
				Title:           b.GetTitle(),
				Author:          "N/A",
				ISBN:            "N/A",
				Category:        b.GetGenre(),
				TotalCopies:     b.GetAvailableCopies(),
				AvailableCopies: b.GetAvailableCopies(),
				Location:        "N/A",
			})
		}

		render.JSON(w, r, Response{
			Response: resp.OK(),
			Books:    result,
		})
	}
}
