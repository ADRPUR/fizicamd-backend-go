package httpapi

import (
	"context"
	"net/http"
	"strings"

	"fizicamd-backend-go/internal/services"
)

type contextKey string

const (
	ctxUserID contextKey = "userID"
	ctxEmail  contextKey = "email"
	ctxRoles  contextKey = "roles"
)

func WithAuth(tokenService services.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				WriteError(w, http.StatusUnauthorized, "Authentication failed")
				return
			}
			tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			token, claims, err := tokenService.ParseToken(tokenStr)
			if err != nil || !token.Valid {
				WriteError(w, http.StatusUnauthorized, "Authentication failed")
				return
			}
			if claims["typ"] != "access" {
				WriteError(w, http.StatusUnauthorized, "Authentication failed")
				return
			}
			userID, _ := claims["sub"].(string)
			email, _ := claims["email"].(string)
			roles := []string{}
			if rawRoles, ok := claims["roles"].([]interface{}); ok {
				for _, r := range rawRoles {
					if s, ok := r.(string); ok {
						roles = append(roles, s)
					}
				}
			}
			ctx := context.WithValue(r.Context(), ctxUserID, userID)
			ctx = context.WithValue(ctx, ctxEmail, email)
			ctx = context.WithValue(ctx, ctxRoles, roles)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CurrentUserID(r *http.Request) string {
	if value, ok := r.Context().Value(ctxUserID).(string); ok {
		return value
	}
	return ""
}

func CurrentRoles(r *http.Request) []string {
	if value, ok := r.Context().Value(ctxRoles).([]string); ok {
		return value
	}
	return nil
}

func RequireRole(role string) func(http.Handler) http.Handler {
	role = strings.ToUpper(role)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, rrole := range CurrentRoles(r) {
				if strings.ToUpper(rrole) == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			WriteError(w, http.StatusForbidden, "Not allowed")
		})
	}
}

func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[strings.ToUpper(role)] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, rrole := range CurrentRoles(r) {
				if allowed[strings.ToUpper(rrole)] {
					next.ServeHTTP(w, r)
					return
				}
			}
			WriteError(w, http.StatusForbidden, "Not allowed")
		})
	}
}
