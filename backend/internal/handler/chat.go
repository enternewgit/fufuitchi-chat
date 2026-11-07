// Package handler は HTTP リクエストを受けて返すだけの層。
// ロジックを書き始めたら service に移すサイン。
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/service"
)

const (
	maxMessageRunes = 1000
	maxIDLen        = 100
)

type ChatHandler struct {
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

type chatRequest struct {
	CharacterID string        `json:"characterId"`
	SessionID   string        `json:"sessionId"`
	Variant     model.Variant `json:"variant"`
	Message     string        `json:"message"`
}

// chatEvent は SSE で送るイベント。type は "delta" / "done" / "error"。
type chatEvent struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Overlap *model.Overlap `json:"overlap,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// Post は POST /api/chat。返答を SSE（text/event-stream）で流す。
func (h *ChatHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Variant == "" {
		req.Variant = model.VariantBase
	}
	if msg := validateChat(req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(ev chatEvent) error {
		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return err
		}
		return rc.Flush()
	}

	res, err := h.svc.Chat(r.Context(), service.ChatInput{
		SessionID:   req.SessionID,
		CharacterID: req.CharacterID,
		Variant:     req.Variant,
		Message:     req.Message,
	}, func(delta string) error {
		return send(chatEvent{Type: "delta", Text: delta})
	})
	if err != nil {
		log.Printf("chat: %v", err)
		send(chatEvent{Type: "error", Error: publicError(err)})
		return
	}
	send(chatEvent{Type: "done", Overlap: res.Overlap})
}

func validateChat(req chatRequest) string {
	switch {
	case req.CharacterID == "" || len(req.CharacterID) > maxIDLen:
		return "characterId が不正です"
	case req.SessionID == "" || len(req.SessionID) > maxIDLen:
		return "sessionId が不正です"
	case !req.Variant.Valid():
		return "variant は base か tuned です"
	case req.Message == "":
		return "message が空です"
	case utf8.RuneCountInString(req.Message) > maxMessageRunes:
		return fmt.Sprintf("message は %d 文字以内にしてください", maxMessageRunes)
	}
	return ""
}

// publicError はユーザーに見せてよいエラーメッセージを返す。
func publicError(err error) string {
	switch {
	case errors.Is(err, service.ErrCharacterNotFound):
		return "キャラが見つかりません"
	case errors.Is(err, service.ErrNoTunedModel):
		return service.ErrNoTunedModel.Error()
	}
	return "返答の生成に失敗しました"
}
