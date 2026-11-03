package model

import "time"

// Character は1キャラ分の設定。
type Character struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SystemPrompt string `json:"systemPrompt"`
	BaseModel    string `json:"baseModel"`  // プロンプト版で使うモデル
	TunedModel   string `json:"tunedModel"` // チューニング済みモデル（空なら未作成）
}

// Message は会話の1発言。
type Message struct {
	SessionID   string    `json:"sessionId"`
	CharacterID string    `json:"characterId"`
	Role        string    `json:"role"` // "user" or "model"
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"createdAt"`
}
