package middleware

import (
	"context"
	"database/sql"
	"strings"

	mdb "office-pong/database"
	"net/http"

	"github.com/gorilla/mux"
)

type ctxKey struct{}

func UserIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(ctxKey{}).(string)
	return uid, ok
}

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

			uid, err := mdb.VerifyUserToken(db, token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, uid)))
		})
	}
}
