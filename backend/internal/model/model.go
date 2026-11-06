package model

import "time"

// Character は1キャラ分の設定。
type Character struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"systemPrompt"`
	BaseModel    string `json:"baseModel"`  // プロンプト版で使うモデル
	TunedModel   string `json:"tunedModel"` // チューニング済みモデル（空なら未作成）
}

// PublicCharacter は一般ユーザーに返すキャラ情報。システムプロンプトは含めない。
type PublicCharacter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HasTuned    bool   `json:"hasTuned"`
}

func (c Character) Public() PublicCharacter {
	return PublicCharacter{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		HasTuned:    c.TunedModel != "",
	}
}

// Variant はどちらのモデルで返答したか。
type Variant string

const (
	VariantBase  Variant = "base"  // プロンプト版
	VariantTuned Variant = "tuned" // チューニング版
)

func (v Variant) Valid() bool {
	return v == VariantBase || v == VariantTuned
}

// Message は会話の1発言。
type Message struct {
	SessionID   string    `json:"sessionId"`
	CharacterID string    `json:"characterId"`
	Variant     Variant   `json:"variant"`
	Role        string    `json:"role"` // "user" or "model"
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"createdAt"`
	// Memorization は返答が学習データとどれくらい一致したか（0〜1）。model の発言のみ。
	Memorization *float64 `json:"memorization,omitempty"`
}

// SessionSummary は管理画面の会話ログ一覧の1行。
type SessionSummary struct {
	SessionID    string    `json:"sessionId"`
	CharacterID  string    `json:"characterId"`
	MessageCount int       `json:"messageCount"`
	FirstAt      time.Time `json:"firstAt"`
	LastAt       time.Time `json:"lastAt"`
}
