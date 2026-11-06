package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
)

type CharacterHandler struct {
	chars repository.CharacterRepository
}

func NewCharacterHandler(chars repository.CharacterRepository) *CharacterHandler {
	return &CharacterHandler{chars: chars}
}

// List は GET /api/characters。公開用なのでシステムプロンプトは返さない。
func (h *CharacterHandler) List(w http.ResponseWriter, r *http.Request) {
	cs, err := h.chars.List(r.Context())
	if err != nil {
		log.Printf("list characters: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := make([]model.PublicCharacter, len(cs))
	for i, c := range cs {
		out[i] = c.Public()
	}
	writeJSON(w, http.StatusOK, out)
}

// Get は GET /api/characters/{id}。
func (h *CharacterHandler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := h.chars.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("get character: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, c.Public())
}
