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
	// TunedSystemPrompt はチューニング版に渡すシステムプロンプト。
	// 空なら渡さない（システムプロンプトなしで学習したので、学習時と同じ条件になる）
	TunedSystemPrompt string `json:"tunedSystemPrompt"`
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
	// Overlap は返答が学習データとどれくらい一致したか。model の発言のみ。
	Overlap *Overlap `json:"overlap,omitempty"`
	// Fallback はチューニング版が上限に達したため、プロンプト版のモデルで返答したかどうか。
	Fallback bool `json:"fallback,omitempty"`
}

// Overlap は返答と学習データのお返事の重なり。
type Overlap struct {
	Longest int  `json:"longest"` // 最も長く連続して一致した文字数（記号・空白は除く）
	Exact   bool `json:"exact"`   // お返事のどれかと丸ごと同じ
}

// SessionSummary は管理画面の会話ログ一覧の1行。
type SessionSummary struct {
	SessionID    string    `json:"sessionId"`
	CharacterID  string    `json:"characterId"`
	MessageCount int       `json:"messageCount"`
	FirstAt      time.Time `json:"firstAt"`
	LastAt       time.Time `json:"lastAt"`
}
