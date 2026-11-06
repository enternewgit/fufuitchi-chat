package repository

import (
	"context"
	"errors"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
)

// ErrNotFound は対象が見つからないときに返す。
var ErrNotFound = errors.New("not found")

// CharacterRepository はキャラ設定の保存先。
// memory / firestore など実装を差し替えられるように interface にしておく。
type CharacterRepository interface {
	List(ctx context.Context) ([]model.Character, error)
	Get(ctx context.Context, id string) (model.Character, error)
	Save(ctx context.Context, c model.Character) error
	Delete(ctx context.Context, id string) error
}

// HistoryQuery は会話履歴の取得条件。
type HistoryQuery struct {
	SessionID   string
	CharacterID string
	Variant     model.Variant
	Limit       int // 直近何件か（0 なら全件）
}

// SessionFilter は管理画面の会話ログ一覧の絞り込み条件。ゼロ値の項目は絞り込まない。
type SessionFilter struct {
	CharacterID string
	From        time.Time // この時刻以降に発言があったセッション
	To          time.Time // この時刻より前に発言があったセッション
}

// MessageRepository は会話ログの保存先。
type MessageRepository interface {
	Append(ctx context.Context, msgs ...model.Message) error
	History(ctx context.Context, q HistoryQuery) ([]model.Message, error)
	// 管理画面用
	ListSessions(ctx context.Context, f SessionFilter) ([]model.SessionSummary, error)
	SessionMessages(ctx context.Context, sessionID string) ([]model.Message, error)
}
