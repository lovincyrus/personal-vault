package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/lovincyrus/personal-vault/internal/store"
)

type contextKey string

const (
	scopeKey      contextKey = "scope"
	sessionAuthKey contextKey = "session_auth"
)

// scopeFromRequest returns the token scope. Session tokens get "*" (full access).
func scopeFromRequest(r *http.Request) string {
	if s, ok := r.Context().Value(scopeKey).(string); ok {
		return s
	}
	return "*"
}

// isSessionAuth returns true if the request was authenticated with a session token.
func isSessionAuth(r *http.Request) bool {
	v, _ := r.Context().Value(sessionAuthKey).(bool)
	return v
}

func sessionRequired(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, "session_required", "this operation requires a session token, not a service token")
}

// securityHeadersMiddleware sets standard security headers on all responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

const maxBodySize = 1 << 20 // 1 MB

// bodySizeMiddleware limits request body size to prevent memory exhaustion.
func bodySizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		}
		next.ServeHTTP(w, r)
	})
}

// authMiddleware extracts the Bearer token and validates it.
// Accepts both session tokens and service tokens.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "missing authorization")
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")

		// Try session token first — full access
		if s.vault.ValidateToken(token) {
			s.vault.TouchSession()
			ctx := context.WithValue(r.Context(), scopeKey, "*")
			ctx = context.WithValue(ctx, sessionAuthKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Try service token — scoped access
		if svcToken, ok := s.vault.ValidateServiceToken(token); ok {
			s.vault.TouchSession()
			s.vault.LogAccess(store.AuditEntry{
				Consumer: svcToken.Consumer,
				Scope:    svcToken.Scope,
				Action:   "api_access",
			})
			ctx := context.WithValue(r.Context(), scopeKey, svcToken.Scope)
			ctx = context.WithValue(ctx, sessionAuthKey, false)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		writeError(w, http.StatusUnauthorized, "unauthenticated", "invalid or expired token")
	})
}
