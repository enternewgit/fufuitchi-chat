package handler

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
)

// 管理者用。middleware.AdminAuth で保護する。
type AdminHandler struct {
	chars    repository.CharacterRepository
	messages repository.MessageRepository
}

func NewAdminHandler(chars repository.CharacterRepository, messages repository.MessageRepository) *AdminHandler {
	return &AdminHandler{chars: chars, messages: messages}
}

var characterIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// ListCharacters は GET /api/admin/characters。システムプロンプトも返す。
func (h *AdminHandler) ListCharacters(w http.ResponseWriter, r *http.Request) {
	cs, err := h.chars.List(r.Context())
	if err != nil {
		h.internal(w, "list characters", err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

// CreateCharacter は POST /api/admin/characters。
func (h *AdminHandler) CreateCharacter(w http.ResponseWriter, r *http.Request) {
	var c model.Character
	if !decodeJSON(w, r, &c) {
		return
	}
	if msg := validateCharacter(c); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if _, err := h.chars.Get(r.Context(), c.ID); err == nil {
		writeError(w, http.StatusConflict, "同じ id のキャラがすでにあります")
		return
	} else if !errors.Is(err, repository.ErrNotFound) {
		h.internal(w, "get character", err)
		return
	}
	if err := h.chars.Save(r.Context(), c); err != nil {
		h.internal(w, "save character", err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// UpdateCharacter は PUT /api/admin/characters/{id}。
func (h *AdminHandler) UpdateCharacter(w http.ResponseWriter, r *http.Request) {
	var c model.Character
	if !decodeJSON(w, r, &c) {
		return
	}
	c.ID = r.PathValue("id")
	if msg := validateCharacter(c); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if _, err := h.chars.Get(r.Context(), c.ID); errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		h.internal(w, "get character", err)
		return
	}
	if err := h.chars.Save(r.Context(), c); err != nil {
		h.internal(w, "save character", err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// DeleteCharacter は DELETE /api/admin/characters/{id}。
func (h *AdminHandler) DeleteCharacter(w http.ResponseWriter, r *http.Request) {
	err := h.chars.Delete(r.Context(), r.PathValue("id"))
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		h.internal(w, "delete character", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListLogs は GET /api/admin/logs?characterId=&from=YYYY-MM-DD&to=YYYY-MM-DD。
// to はその日を含む。
func (h *AdminHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repository.SessionFilter{CharacterID: q.Get("characterId")}
	var err error
	if s := q.Get("from"); s != "" {
		if f.From, err = time.ParseInLocation(time.DateOnly, s, time.Local); err != nil {
			writeError(w, http.StatusBadRequest, "from は YYYY-MM-DD で指定してください")
			return
		}
	}
	if s := q.Get("to"); s != "" {
		if f.To, err = time.ParseInLocation(time.DateOnly, s, time.Local); err != nil {
			writeError(w, http.StatusBadRequest, "to は YYYY-MM-DD で指定してください")
			return
		}
		f.To = f.To.AddDate(0, 0, 1)
	}
	sessions, err := h.messages.ListSessions(r.Context(), f)
	if err != nil {
		h.internal(w, "list sessions", err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

// GetLog は GET /api/admin/logs/{sessionId}。
func (h *AdminHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	msgs, err := h.messages.SessionMessages(r.Context(), r.PathValue("sessionId"))
	if err != nil {
		h.internal(w, "session messages", err)
		return
	}
	if len(msgs) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

// DeleteLog は DELETE /api/admin/logs/{sessionId}。会話の発言をすべて消す。
// 前の会話の記憶にも使われなくなる。
func (h *AdminHandler) DeleteLog(w http.ResponseWriter, r *http.Request) {
	err := h.messages.DeleteSession(r.Context(), r.PathValue("sessionId"))
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		h.internal(w, "delete session", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) internal(w http.ResponseWriter, what string, err error) {
	log.Printf("admin: %s: %v", what, err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func validateCharacter(c model.Character) string {
	switch {
	case !characterIDPattern.MatchString(c.ID):
		return "id は英小文字・数字・-・_ の64文字以内にしてください"
	case strings.TrimSpace(c.Name) == "":
		return "name が空です"
	case strings.TrimSpace(c.BaseModel) == "":
		return "baseModel が空です"
	}
	return ""
}
