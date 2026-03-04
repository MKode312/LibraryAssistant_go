package register

import (
	ssogrpc "LibAssistant_api/internal/clients/sso/grpc"
	studentsgrpc "LibAssistant_api/internal/clients/students/grpc"
	resp "LibAssistant_api/internal/lib/api/response"
	"LibAssistant_api/internal/lib/convert"
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
	FullName string `json:"fullName" validate:"required"`
	Grade    int32  `json:"grade" validate:"required"`
	Letter   string `json:"letter" validate:"required"`
}

type Response struct {
	resp.Response
	StudentID string `json:"studentID"`
}

func New(ctx context.Context, log *slog.Logger, ssoClient *ssogrpc.Client, studentsClient *studentsgrpc.Client) http.HandlerFunc {
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

		log.Info("user registered", slog.String("id", userID))

		studentID, err := studentsClient.CreateStudent(ctx, req.FullName, req.Grade, req.Letter, userID)
		if err != nil {
			if errors.Is(err, studentsgrpc.ErrInvalidRequest) {
				log.Error("invalid credentials", sl.Err(err))
				success, delErr := ssoClient.DeleteUserByID(ctx, userID)
				if delErr != nil {
					if errors.Is(delErr, ssogrpc.ErrInvalidCredentials) {
						log.Error("invalid credentials")
						w.WriteHeader(http.StatusBadRequest)
						render.JSON(w, r, resp.Error("Invalid request"))
						return
					}
					if errors.Is(delErr, ssogrpc.ErrUserNotFound) {
						log.Error("user not found")
						w.WriteHeader(http.StatusConflict)
						render.JSON(w, r, resp.Error("User not found in sso"))
						return
					}

					log.Error("failed to delete user", sl.Err(delErr))
					w.WriteHeader(http.StatusInternalServerError)
					render.JSON(w, r, resp.Error("Unknown error"))
					return
				}
				if success {
					log.Info("user was deleted, rollback successful")
				} else {
					log.Warn("user was not deleted, rollback unsuccessful")
				}
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid request" + ":" + convert.StudentsServiceErrorToHTTPResponseError(err)))
				return
			}

			if errors.Is(err, studentsgrpc.ErrStudentExists) {
				log.Error("student exists")
				success, err := ssoClient.DeleteUserByID(ctx, userID)
				if err != nil {
					if errors.Is(err, ssogrpc.ErrInvalidCredentials) {
						log.Error("invalid credentials")
						w.WriteHeader(http.StatusBadRequest)
						render.JSON(w, r, resp.Error("Invalid request"))
						return
					}
					if errors.Is(err, ssogrpc.ErrUserNotFound) {
						log.Error("user not found")
						w.WriteHeader(http.StatusConflict)
						render.JSON(w, r, resp.Error("User not found in sso"))
						return
					}

					log.Error("failed to delete user", sl.Err(err))
					w.WriteHeader(http.StatusInternalServerError)
					render.JSON(w, r, resp.Error("Unknown error"))
					return
				}
				if success {
					log.Info("user was deleted, rollback successful")
				} else {
					log.Warn("user was not deleted, rollback unsuccessful")
				}
				w.WriteHeader(http.StatusConflict)
				render.JSON(w, r, resp.Error("Student profile already exists"))
				return
			}

			if errors.Is(err, studentsgrpc.ErrInternal) {
				log.Error("internal error", sl.Err(err))
				success, err := ssoClient.DeleteUserByID(ctx, userID)
				if err != nil {
					if errors.Is(err, ssogrpc.ErrInvalidCredentials) {
						log.Error("invalid credentials")
						w.WriteHeader(http.StatusBadRequest)
						render.JSON(w, r, resp.Error("Invalid request: " + err.Error()))
						return
					}
					if errors.Is(err, ssogrpc.ErrUserNotFound) {
						log.Error("user not found")
						w.WriteHeader(http.StatusConflict)
						render.JSON(w, r, resp.Error("User not found in sso"))
						return
					}

					log.Error("failed to delete user", sl.Err(err))
					w.WriteHeader(http.StatusInternalServerError)
					render.JSON(w, r, resp.Error("Unknown error"))
					return
				}
				if success {
					log.Info("user was deleted, rollback successful")
				} else {
					log.Warn("user was not deleted, rollback unsuccessful")
				}
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Unknown error"))
				return
			}

			log.Error("failed to create student", sl.Err(err))
			success, err := ssoClient.DeleteUserByID(ctx, userID)
			if err != nil {
				if errors.Is(err, ssogrpc.ErrInvalidCredentials) {
					log.Error("invalid credentials")
					w.WriteHeader(http.StatusBadRequest)
					render.JSON(w, r, resp.Error("Invalid request"))
					return
				}
				if errors.Is(err, ssogrpc.ErrUserNotFound) {
					log.Error("user not found")
					w.WriteHeader(http.StatusConflict)
					render.JSON(w, r, resp.Error("User not found in sso"))
					return
				}

				log.Error("failed to delete user", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Unknown error"))
				return
			}
			if success {
				log.Info("user was deleted, rollback successful")
			} else {
				log.Warn("user was not deleted, rollback unsuccessful")
			}
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		log.Info("user was registered and the student was created successfully")

		http.SetCookie(w, &http.Cookie{
			Name: "is_admin",
			Value: "false",
			Path: "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		log.Info("cookie was set")

		w.WriteHeader(http.StatusCreated)

		render.JSON(w, r, Response{
			Response:  resp.OK(),
			StudentID: studentID,
		})
	}
}

