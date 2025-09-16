package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/diagnosis/luxsuv-bookings/internal/platform/auth"
)

type ctxKey string

const CtxClaims ctxKey = "claims"

func RequireJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}
		raw := strings.TrimPrefix(authz, "Bearer ")
		claims, err := auth.Parse(raw)
		if err != nil {
			http.Error(w, "invalid authorization token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), CtxClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Claims(r *http.Request) *auth.Claims {
	v := r.Context().Value(CtxClaims)
	if v == nil {
		return nil
	}
	return v.(*auth.Claims)
}
