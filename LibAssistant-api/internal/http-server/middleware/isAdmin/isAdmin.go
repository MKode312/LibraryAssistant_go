package MWIsadmin

import (
	resp "LibAssistant_api/internal/lib/api/response"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	log = log.With(
		slog.String("component", "middleware/isAdmin"),
	)

	log.Info("is_admin middleware enabled")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("is_admin")
			if err != nil {
				log.Error("cookie not found")
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, resp.Error("User is not an admin"))
				return
			}

			isAdmin := cookie.Value

			if isAdmin == "false" {
				log.Warn("forbidden to use this endpoint, user is not an admin")
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, resp.Error("User is not an admin"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
