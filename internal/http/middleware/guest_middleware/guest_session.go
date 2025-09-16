package guest_middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/diagnosis/luxsuv-bookings/internal/platform/auth"
)

type ctxKey string

const CtxClaims ctxKey = "claims"

func RequireGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.URL.Query().Get("session_token")
		if tok == "" {
			authz := r.Header.Get("Authorization")
			if strings.HasPrefix(authz, "Bearer ") {
				tok = strings.TrimPrefix(authz, "Bearer ")
			}
		}
		if tok == "" {
			http.Error(w, "session_token is required", http.StatusUnauthorized)
			return
		}
		claims, err := auth.Parse(tok)

		if err != nil || claims.Role != "guest" {
			http.Error(w, "invalid session_token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), CtxClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func Claims(r *http.Request) *auth.Claims {
	if v := r.Context().Value(CtxClaims); v != nil {
		if c, ok := v.(*auth.Claims); ok {
			return c
		}
	}
	return nil

}
func OptionalGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.URL.Query().Get("session_token")
		if tok == "" {
			if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
				tok = strings.TrimPrefix(authz, "Bearer ")
			}
		}
		if tok != "" {
			if claims, err := auth.Parse(tok); err == nil && claims.Role == "guest" {
				ctx := context.WithValue(r.Context(), CtxClaims, claims)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}
