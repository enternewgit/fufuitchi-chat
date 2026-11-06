// Package service はアプリの処理の本体。handler からはここを呼ぶ。
package service

import (
	"context"
	"errors"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/llm"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
)

// historyLimit はモデルに渡す直近の発言数。
const historyLimit = 20

var (
	ErrCharacterNotFound = errors.New("character not found")
	ErrNoTunedModel      = errors.New("このキャラのチューニング済みモデルはまだありません")
)

type ChatService struct {
	chars    repository.CharacterRepository
	messages repository.MessageRepository
	llm      llm.Generator
	memo     *Memorizer
	now      func() time.Time
}

func NewChatService(chars repository.CharacterRepository, messages repository.MessageRepository, gen llm.Generator, memo *Memorizer) *ChatService {
	return &ChatService{chars: chars, messages: messages, llm: gen, memo: memo, now: time.Now}
}

type ChatInput struct {
	SessionID   string
	CharacterID string
	Variant     model.Variant
	Message     string
}

type ChatResult struct {
	Reply        string
	Memorization *float64
}

// Chat は会話を1往復進める。返答は onDelta に少しずつ渡す。
//  1. キャラ設定を取得
//  2. セッションの直近の履歴を取得
//  3. モデルを呼び出し、返答をストリーミング
//  4. 丸暗記率を計算
//  5. ユーザー発言と返答を保存
func (s *ChatService) Chat(ctx context.Context, in ChatInput, onDelta func(string) error) (ChatResult, error) {
	c, err := s.chars.Get(ctx, in.CharacterID)
	if errors.Is(err, repository.ErrNotFound) {
		return ChatResult{}, ErrCharacterNotFound
	}
	if err != nil {
		return ChatResult{}, err
	}

	modelID := c.BaseModel
	if in.Variant == model.VariantTuned {
		if c.TunedModel == "" {
			return ChatResult{}, ErrNoTunedModel
		}
		modelID = c.TunedModel
	}

	history, err := s.messages.History(ctx, repository.HistoryQuery{
		SessionID:   in.SessionID,
		CharacterID: in.CharacterID,
		Variant:     in.Variant,
		Limit:       historyLimit,
	})
	if err != nil {
		return ChatResult{}, err
	}

	userAt := s.now()
	reply, err := s.llm.Stream(ctx, llm.Request{
		Model:        modelID,
		SystemPrompt: c.SystemPrompt,
		History:      history,
		Message:      in.Message,
	}, onDelta)
	if err != nil {
		return ChatResult{}, err
	}

	var memo *float64
	if s.memo != nil {
		if score, ok := s.memo.Score(reply); ok {
			memo = &score
		}
	}

	base := model.Message{SessionID: in.SessionID, CharacterID: in.CharacterID, Variant: in.Variant}
	user, bot := base, base
	user.Role, user.Content, user.CreatedAt = "user", in.Message, userAt
	bot.Role, bot.Content, bot.CreatedAt, bot.Memorization = "model", reply, s.now(), memo
	// 返答の送信後に保存するので、クライアントが切断していても保存できるよう ctx を切り離す
	if err := s.messages.Append(context.WithoutCancel(ctx), user, bot); err != nil {
		return ChatResult{}, err
	}

	return ChatResult{Reply: reply, Memorization: memo}, nil
}
