// Package memory はメモリ上に保存する実装。再起動するとデータは消える。
// 最初はこれで動かして、必要になったら firestore に差し替える。
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
)

// CharacterRepo は repository.CharacterRepository のメモリ実装。
type CharacterRepo struct {
	mu    sync.RWMutex
	chars map[string]model.Character
}

func NewCharacterRepo() *CharacterRepo {
	return &CharacterRepo{chars: map[string]model.Character{}}
}

// LoadDir は dir 内の *.json（1ファイル1キャラ）を読み込む。
func (r *CharacterRepo) LoadDir(dir string) (int, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return 0, err
		}
		var c model.Character
		if err := json.Unmarshal(b, &c); err != nil {
			return 0, fmt.Errorf("%s: %w", p, err)
		}
		if c.ID == "" {
			return 0, fmt.Errorf("%s: id is empty", p)
		}
		r.chars[c.ID] = c
	}
	return len(paths), nil
}

func (r *CharacterRepo) List(ctx context.Context) ([]model.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]model.Character, 0, len(r.chars))
	for _, c := range r.chars {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *CharacterRepo) Get(ctx context.Context, id string) (model.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.chars[id]
	if !ok {
		return model.Character{}, repository.ErrNotFound
	}
	return c, nil
}

func (r *CharacterRepo) Save(ctx context.Context, c model.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chars[c.ID] = c
	return nil
}

func (r *CharacterRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.chars[id]; !ok {
		return repository.ErrNotFound
	}
	delete(r.chars, id)
	return nil
}

// MessageRepo は repository.MessageRepository のメモリ実装。
// 発言は追加順に並んでいる前提で扱う。
type MessageRepo struct {
	mu   sync.RWMutex
	msgs []model.Message
}

func NewMessageRepo() *MessageRepo {
	return &MessageRepo{}
}

func (r *MessageRepo) Append(ctx context.Context, msgs ...model.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, msgs...)
	return nil
}

func (r *MessageRepo) History(ctx context.Context, q repository.HistoryQuery) ([]model.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []model.Message
	for _, m := range r.msgs {
		if m.SessionID == q.SessionID && m.CharacterID == q.CharacterID && m.Variant == q.Variant {
			out = append(out, m)
		}
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[len(out)-q.Limit:]
	}
	return out, nil
}

func (r *MessageRepo) UserHistory(ctx context.Context, q repository.UserHistoryQuery) ([]model.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []model.Message
	for _, m := range r.msgs {
		if m.UserID == q.UserID && m.CharacterID == q.CharacterID && m.Variant == q.Variant && m.SessionID != q.ExcludeSessionID {
			out = append(out, m)
		}
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[len(out)-q.Limit:]
	}
	return out, nil
}

func (r *MessageRepo) ListSessions(ctx context.Context, f repository.SessionFilter) ([]model.SessionSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	byID := map[string]*model.SessionSummary{}
	for _, m := range r.msgs {
		if f.CharacterID != "" && m.CharacterID != f.CharacterID {
			continue
		}
		s, ok := byID[m.SessionID]
		if !ok {
			s = &model.SessionSummary{SessionID: m.SessionID, CharacterID: m.CharacterID, FirstAt: m.CreatedAt}
			byID[m.SessionID] = s
		}
		s.MessageCount++
		s.LastAt = m.CreatedAt
	}
	out := make([]model.SessionSummary, 0, len(byID))
	for _, s := range byID {
		if !f.From.IsZero() && s.LastAt.Before(f.From) {
			continue
		}
		if !f.To.IsZero() && !s.FirstAt.Before(f.To) {
			continue
		}
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastAt.After(out[j].LastAt) })
	return out, nil
}

func (r *MessageRepo) SessionMessages(ctx context.Context, sessionID string) ([]model.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []model.Message
	for _, m := range r.msgs {
		if m.SessionID == sessionID {
			out = append(out, m)
		}
	}
	return out, nil
}
