package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/lovincyrus/personal-vault/internal/vault"
)

func scopeDenied(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, "scope_exceeded", "token scope does not allow access to this field")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, constraint, msg string) {
	writeJSON(w, status, map[string]string{"error": msg, "constraint": constraint})
}

// POST /vault/unlock
func (s *Server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if !s.unlockLimit.allow() {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many unlock attempts, try again later")
		return
	}

	var req struct {
		Password  string `json:"password"`
		SecretKey string `json:"secret_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}
	if req.Password == "" || req.SecretKey == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "password and secret_key required")
		return
	}

	token, err := s.vault.Unlock(req.Password, req.SecretKey)
	if err != nil {
		switch err {
		case vault.ErrWrongPassword:
			writeError(w, http.StatusUnauthorized, "unauthenticated", "wrong password or secret key")
		case vault.ErrAlreadyUnlocked:
			writeError(w, http.StatusConflict, "conflict", "vault is already unlocked")
		case vault.ErrNotInitialized:
			writeError(w, http.StatusPreconditionFailed, "not_initialized", "vault is not initialized")
		default:
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// POST /vault/lock
func (s *Server) handleLock(w http.ResponseWriter, r *http.Request) {
	if !isSessionAuth(r) {
		sessionRequired(w)
		return
	}
	s.vault.Lock()
	writeJSON(w, http.StatusOK, map[string]string{"status": "locked"})
}

// GET /vault/status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.vault.Status()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// GET /vault/fields
func (s *Server) handleListFields(w http.ResponseWriter, r *http.Request) {
	fields, err := s.vault.List()
	if err != nil {
		handleVaultError(w, err)
		return
	}
	scope := scopeFromRequest(r)
	allowed := make([]vault.FieldInfo, 0, len(fields))
	for _, f := range fields {
		if vault.ScopeAllows(scope, f.ID) {
			allowed = append(allowed, f)
		}
	}
	writeJSON(w, http.StatusOK, allowed)
}

// GET /vault/fields/{id...}
func (s *Server) handleGetField(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := vault.ValidateFieldID(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), id) {
		scopeDenied(w)
		return
	}
	field, err := s.vault.Get(id)
	if err != nil {
		handleVaultError(w, err)
		return
	}
	if field == nil {
		writeError(w, http.StatusNotFound, "not_found", "field not found")
		return
	}
	writeJSON(w, http.StatusOK, field)
}

// PUT /vault/fields/{id...}
func (s *Server) handleSetField(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := vault.ValidateFieldID(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), id) {
		scopeDenied(w)
		return
	}
	var req struct {
		Value       string `json:"value"`
		Sensitivity string `json:"sensitivity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}
	if req.Value == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "value required")
		return
	}

	// Apply schema default sensitivity when none provided
	if req.Sensitivity == "" {
		req.Sensitivity = vault.DefaultSensitivity(id)
	}

	if err := s.vault.Set(id, req.Value, req.Sensitivity); err != nil {
		handleVaultError(w, err)
		return
	}

	resp := map[string]any{"status": "ok"}
	if suggestion := vault.SuggestCanonical(id); suggestion != nil {
		resp["suggestion"] = suggestion
	}
	writeJSON(w, http.StatusOK, resp)
}

// GET /vault/schema
func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, vault.RecommendedSchema)
}

// DELETE /vault/fields/{id...}
func (s *Server) handleDeleteField(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := vault.ValidateFieldID(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), id) {
		scopeDenied(w)
		return
	}
	if err := s.vault.Delete(id); err != nil {
		handleVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GET /vault/fields/category/{category}
func (s *Server) handleGetByCategory(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("category")
	if !vault.ValidCategoryName(category) {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid category name: only alphanumeric, underscore, hyphen allowed")
		return
	}
	scope := scopeFromRequest(r)
	if !vault.ScopeAllowsCategory(scope, category) {
		scopeDenied(w)
		return
	}
	fields, err := s.vault.GetByCategory(category)
	if err != nil {
		handleVaultError(w, err)
		return
	}
	// Filter to only fields allowed by scope (handles exact field patterns)
	allowed := make([]vault.FieldInfo, 0, len(fields))
	for _, f := range fields {
		if vault.ScopeAllows(scope, f.ID) {
			allowed = append(allowed, f)
		}
	}
	writeJSON(w, http.StatusOK, allowed)
}

// GET /vault/context
func (s *Server) handleGetContext(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.vault.GetContext()
	if err != nil {
		handleVaultError(w, err)
		return
	}
	scope := scopeFromRequest(r)
	if scope != "*" {
		filtered := &vault.ContextBundle{Categories: make(map[string][]vault.FieldInfo)}
		for cat, fields := range ctx.Categories {
			for _, f := range fields {
				if vault.ScopeAllows(scope, f.ID) {
					filtered.Categories[cat] = append(filtered.Categories[cat], f)
				}
			}
		}
		ctx = filtered
	}
	writeJSON(w, http.StatusOK, ctx)
}

// GET /vault/audit
func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	if !isSessionAuth(r) {
		sessionRequired(w)
		return
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	entries, err := s.vault.AuditLog(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// PUT /vault/sensitivity/{id...}
func (s *Server) handleSetSensitivity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := vault.ValidateFieldID(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), id) {
		scopeDenied(w)
		return
	}
	var req struct {
		Tier string `json:"tier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}
	if req.Tier == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "tier required")
		return
	}

	if err := s.vault.SetSensitivity(id, req.Tier); err != nil {
		handleVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /vault/tokens/service
func (s *Server) handleCreateServiceToken(w http.ResponseWriter, r *http.Request) {
	if !isSessionAuth(r) {
		sessionRequired(w)
		return
	}
	var req struct {
		Consumer string `json:"consumer"`
		Scope    string `json:"scope"`
		TTL      string `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}
	if req.Consumer == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "consumer required")
		return
	}
	if req.Scope == "" {
		req.Scope = "*"
	}

	ttl := 365 * 24 * time.Hour // default 1 year
	if req.TTL != "" {
		parsed, err := time.ParseDuration(req.TTL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid ttl duration")
			return
		}
		ttl = parsed
	}

	token, err := s.vault.CreateServiceToken(req.Consumer, req.Scope, ttl)
	if err != nil {
		handleVaultError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"token":      token,
		"expires_at": time.Now().Add(ttl).UTC().Format(time.RFC3339),
	})
}

// GET /vault/tokens/service
func (s *Server) handleListServiceTokens(w http.ResponseWriter, r *http.Request) {
	if !isSessionAuth(r) {
		sessionRequired(w)
		return
	}
	tokens, err := s.vault.ListServiceTokens()
	if err != nil {
		handleVaultError(w, err)
		return
	}

	type tokenInfo struct {
		TokenPrefix string `json:"token_prefix"`
		Consumer    string `json:"consumer"`
		Scope       string `json:"scope"`
		ExpiresAt   string `json:"expires_at"`
		CreatedAt   string `json:"created_at"`
	}

	result := make([]tokenInfo, len(tokens))
	for i, t := range tokens {
		// TokenStr is now a hash; show first 8 chars for identification
		hashPrefix := t.TokenStr
		if len(hashPrefix) > 8 {
			hashPrefix = hashPrefix[:8] + "..."
		}
		result[i] = tokenInfo{
			TokenPrefix: hashPrefix,
			Consumer:    t.Consumer,
			Scope:       t.Scope,
			ExpiresAt:   t.ExpiresAt.UTC().Format(time.RFC3339),
			CreatedAt:   t.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// DELETE /vault/tokens/service/{token}
func (s *Server) handleRevokeServiceToken(w http.ResponseWriter, r *http.Request) {
	if !isSessionAuth(r) {
		sessionRequired(w)
		return
	}
	prefix := r.PathValue("token")
	if prefix == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "token prefix required")
		return
	}

	n, err := s.vault.RevokeServiceToken(prefix)
	if err != nil {
		handleVaultError(w, err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "not_found", "no matching token found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "revoked", "count": n})
}

func handleVaultError(w http.ResponseWriter, err error) {
	switch err {
	case vault.ErrLocked:
		writeError(w, http.StatusForbidden, "vault_locked", "vault is locked")
	case vault.ErrAlreadyUnlocked:
		writeError(w, http.StatusConflict, "conflict", "vault is already unlocked")
	case vault.ErrNotInitialized:
		writeError(w, http.StatusPreconditionFailed, "not_initialized", "vault is not initialized")
	case vault.ErrInvalidTier:
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
