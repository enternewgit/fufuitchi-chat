package repository

import (
	"context"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
)

// CharacterRepository はキャラ設定の保存先。
// memory / firestore など実装を差し替えられるように interface にしておく。
type CharacterRepository interface {
	List(ctx context.Context) ([]model.Character, error)
	Get(ctx context.Context, id string) (model.Character, error)
	Save(ctx context.Context, c model.Character) error
	Delete(ctx context.Context, id string) error
}

// MessageRepository は会話ログの保存先。
type MessageRepository interface {
	Append(ctx context.Context, m model.Message) error
	History(ctx context.Context, sessionID string, limit int) ([]model.Message, error)
	// TODO: 管理画面用の一覧取得（キャラ・日付での絞り込み）
}
