package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ContextKeySubject contextKey = "auth.subject"
	ContextKeyRole    contextKey = "auth.role"
)

// Verifier checks a token and returns subject and role
type Verifier interface {
	Verify(ctx context.Context, token string) (subject string, role string, err error)
}

func AuthMiddleware(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			sub, role, err := verifier.Verify(r.Context(), token)
			if err != nil {
				// Map detailed error if needed, for now 401
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Set in Context
			ctx := context.WithValue(r.Context(), ContextKeySubject, sub)
			ctx = context.WithValue(ctx, ContextKeyRole, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Helpers to get data from context
func GetSubject(ctx context.Context) string {
	s, _ := ctx.Value(ContextKeySubject).(string)
	return s
}

func GetRole(ctx context.Context) string {
	r, _ := ctx.Value(ContextKeyRole).(string)
	return r
}
