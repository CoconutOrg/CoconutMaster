package middlewares

import (
	"net/http"
	"strings"

	repo "github.com/CoconutOrg/CoconutMaster/internal/adapters/sqlc"
	"github.com/CoconutOrg/CoconutMaster/internal/services/auth"
)

func JwtAuthentication(secret []byte, q *repo.Queries) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.Header.Get("Authorization")

			if tokenString == "" {
				w.Write([]byte("Missing authorization header!"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !strings.HasPrefix(tokenString, "Bearer ") {
				w.Write([]byte("Invalid token format"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			tokenString, _ = strings.CutPrefix(tokenString, "Bearer ")
			if tokenString == "" {
				w.Write([]byte("Invalid token format"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			token, err := auth.ParseToken(secret, tokenString)
			if err != nil {
				w.Write([]byte("Invalid token"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			claims, ok := auth.GetClaims(token)
			if !ok {
				w.Write([]byte("Invalid token"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			user, err := q.GetUserByID(r.Context(), int64(claims["userID"].(float64)))
			if err != nil {
				w.Write([]byte("Invalid token"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			ok, err = auth.ValidateClaims(token, &user)
			if !ok || err != nil {
				w.Write([]byte("Invalid token"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
