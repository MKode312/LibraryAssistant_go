package register

import (
	ssogrpc "LibAssistant_api/internal/clients/sso/grpc"
	"LibAssistant_api/internal/lib/api/response"
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
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type Response struct {
	resp.Response
	UserID int64  `json:"userID"`
}

func New(ctx context.Context, log *slog.Logger, ssoClient *ssogrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Auth.Register.New"

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

		email := req.Email
		password := req.Password

		userID, err := ssoClient.RegisterNewUser(ctx, email, password)
		if err != nil {
			if errors.Is(err, ssogrpc.ErrInvalidCredentials) {
				log.Error("invalid credentials")

				w.WriteHeader(http.StatusBadRequest)

				render.JSON(w, r, resp.Error("Invalid email or password"))

				return
			}

			if errors.Is(err, ssogrpc.ErrUserExists) {
				log.Error("user already exists")

				w.WriteHeader(http.StatusConflict)

				render.JSON(w, r, resp.Error("You cannot register the existing user"))

				return
			}

			log.Error("failed to register new user", sl.Err(err))

			w.WriteHeader(http.StatusInternalServerError)

			render.JSON(w, r, resp.Error("Unknown error"))

			return
		}

		log.Info("user registered", slog.Int64("id", userID))

		w.WriteHeader(http.StatusCreated)

		render.JSON(w, r, Response{
			Response: resp.OK(),
			UserID: userID,
		})
	}
}
