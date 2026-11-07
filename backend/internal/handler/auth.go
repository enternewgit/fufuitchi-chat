package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/auth"
)

// AuthHandler は管理者の Google ログインを扱う。
type AuthHandler struct {
	verifier auth.Verifier
	allow    auth.Allowlist
	sessions auth.SessionStore
}

func NewAuthHandler(v auth.Verifier, allow auth.Allowlist, sessions auth.SessionStore) *AuthHandler {
	return &AuthHandler{verifier: v, allow: allow, sessions: sessions}
}

type loginResponse struct {
	Token string `json:"token"`
	auth.Session
}

// Google は POST /api/auth/google。ID トークンを検証し、許可したメールアドレスならセッションを発行する。
func (h *AuthHandler) Google(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Credential string `json:"credential"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Credential == "" {
		writeError(w, http.StatusBadRequest, "credential が空です")
		return
	}

	id, err := h.verifier.Verify(r.Context(), req.Credential)
	if err != nil {
		log.Printf("auth: verify: %v", err)
		writeError(w, http.StatusUnauthorized, "Google のログイン情報を確認できませんでした")
		return
	}
	if !id.EmailVerified || !h.allow.Allowed(id.Email) {
		log.Printf("auth: denied %q (verified=%v)", id.Email, id.EmailVerified)
		writeError(w, http.StatusForbidden, "このアカウントには管理者の権限がありません")
		return
	}

	token, sess, err := h.sessions.Create(r.Context(), id.Email)
	if err != nil {
		log.Printf("auth: create session: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.Printf("auth: login %q", id.Email)
	writeJSON(w, http.StatusOK, loginResponse{Token: token, Session: sess})
}

// Logout は POST /api/auth/logout。セッションを削除する。
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		if err := h.sessions.Delete(r.Context(), token); err != nil {
			log.Printf("auth: delete session: %v", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// Me は GET /api/admin/me。ログイン中の管理者を返す（AdminAuth の内側で使う）。
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	sess, _ := auth.FromContext(r.Context())
	writeJSON(w, http.StatusOK, sess)
}
