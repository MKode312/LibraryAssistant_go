package isAdmin

import (
	ssogrpc "LibAssistant_api/internal/clients/sso/grpc"
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
	UserID int64 `json:"userID" validate:"required"`
}

type Response struct {
	resp.Response
	IsAdmin bool `json:"isAdmin"`
}

func New(ctx context.Context, log *slog.Logger, ssoClient *ssogrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Auth.IsAdmin.New"

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

		userID := req.UserID

		isAdmin, err := ssoClient.IsAdmin(ctx, userID)
		if err != nil {
			if errors.Is(err, ssogrpc.ErrInvalidCredentials) {
				log.Error("invalid credentials")

				w.WriteHeader(http.StatusBadRequest)

				render.JSON(w, r, resp.Error("Invalid email or password"))

				return
			}

			if errors.Is(err, ssogrpc.ErrUserNotFound) {
				log.Error("user not found")

				w.WriteHeader(http.StatusUnprocessableEntity)

				render.JSON(w, r, resp.Error("Not found"))

				return
			}

			if errors.Is(err, ssogrpc.ErrInternal) {
				log.Error("internal error")

				w.WriteHeader(http.StatusInternalServerError)

				render.JSON(w, r, resp.Error("Unknown internal error"))

				return
			}

			log.Error("failed to check if user is admin", sl.Err(err))

			w.WriteHeader(http.StatusInternalServerError)

			render.JSON(w, r, resp.Error("Unknown error"))

			return
		}

		if isAdmin {
			http.SetCookie(w, &http.Cookie{
				Name:     "isAdmin",
				Value:    "true",
				HttpOnly: true,
				Path:     "/auth/isAdmin",
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			})

			log.Info("checked if user is admin")

			w.WriteHeader(http.StatusAccepted)

			render.JSON(w, r, Response{
				Response: resp.OK(),
				IsAdmin:  isAdmin,
			})
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:     "isAdmin",
				Value:    "false",
				HttpOnly: true,
				Path:     "/auth/isAdmin",
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			})
			log.Info("checked if user is admin")

			w.WriteHeader(http.StatusForbidden)

			render.JSON(w, r, Response{
				Response: resp.OK(),
				IsAdmin:  isAdmin,
			})
		}
	}
}
