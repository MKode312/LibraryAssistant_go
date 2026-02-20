package MWJwt

import (
	jwtValidation "LibAssistant_api/internal/lib/jwt"
	"LibAssistant_api/internal/lib/logger/sl"
	"log/slog"
	"net/http"
)

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	log = log.With(
		slog.String("component", "middleware/jwt"),
	)

	log.Info("jwt validation middleware enabled")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				log.Error("cookie not found")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			token := cookie.Value

			if err := jwtValidation.ValidateToken(token); err != nil {
				log.Error("validation jwt error", sl.Err(err))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
