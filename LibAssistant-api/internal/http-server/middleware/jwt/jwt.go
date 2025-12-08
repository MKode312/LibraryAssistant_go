package MWJwt

import (
	jwtValidation "LibAssistant_api/internal/lib/jwt"
	"log/slog"
	"net/http"
)

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log = log.With(
			slog.String("component", "middleware/jwt"),
		)

		log.Info("jwt validation middleware enabled")

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				log.Error("cookie not found")
				return
			}

			token := cookie.Value

			if err := jwtValidation.ValidateToken(token); err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}

			next.ServeHTTP(w, r)
		})
		
	}
}
