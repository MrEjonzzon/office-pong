package middleware

import (
	"database/sql"
	"strings"

	mdb "office-pong/database"
	"net/http"

	"github.com/gorilla/mux"
)

func AuthMiddleware(db *sql.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, "no token provided", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			token := strings.TrimSpace(parts[1])

			if token == "" || token == "undefined" {
				http.Error(w, "no token provided", http.StatusUnauthorized)
				return
			}

			if _, err := mdb.VerifyUserToken(db, token); err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
