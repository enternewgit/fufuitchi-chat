// Package service はアプリの処理の本体。handler からはここを呼ぶ。
package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/llm"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
)

// モデルに渡す直近の発言数。
// チューニング版はチャットの本体なので、ふつうの会話なら全部覚えていられる量にする。
// 上限をなくすと、同じ会話で話し続けるほど1回の呼び出しが長くなり、料金を狙った攻撃に使えてしまう
const (
	baseHistoryLimit  = 20
	tunedHistoryLimit = 100
	// pastSessionLimit は、同じユーザーの前の会話から引き継ぐ発言数（10往復分）
	pastSessionLimit = 20
)

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
	UserID      string // 空なら前の会話は引き継がない
	SessionID   string
	CharacterID string
	Variant     model.Variant
	Message     string
}

type ChatResult struct {
	Reply    string
	Overlap  *model.Overlap
	Fallback bool // チューニング版が上限に達したため、プロンプト版で返答した
}

// Chat は会話を1往復進める。返答は onDelta に少しずつ渡す。
//  1. キャラ設定を取得
//  2. セッションの直近の履歴を取得
//  3. モデルを呼び出し、返答をストリーミング
//  4. 学習データとの一致を計算
//  5. ユーザー発言と返答を保存
func (s *ChatService) Chat(ctx context.Context, in ChatInput, onDelta func(string) error) (ChatResult, error) {
	c, err := s.chars.Get(ctx, in.CharacterID)
	if errors.Is(err, repository.ErrNotFound) {
		return ChatResult{}, ErrCharacterNotFound
	}
	if err != nil {
		return ChatResult{}, err
	}

	modelID, systemPrompt, limit := c.BaseModel, c.SystemPrompt, baseHistoryLimit
	if in.Variant == model.VariantTuned {
		if c.TunedModel == "" {
			return ChatResult{}, ErrNoTunedModel
		}
		modelID, systemPrompt, limit = c.TunedModel, c.TunedSystemPrompt, tunedHistoryLimit
	}

	history, err := s.messages.History(ctx, repository.HistoryQuery{
		SessionID:   in.SessionID,
		CharacterID: in.CharacterID,
		Variant:     in.Variant,
		Limit:       limit,
	})
	if err != nil {
		return ChatResult{}, err
	}
	// 同じユーザーの前の会話の直近の発言を、今の会話の前に足す。
	// 「新しい会話」を始めても、前に話したことを覚えているようにするため
	if in.UserID != "" {
		past, err := s.messages.UserHistory(ctx, repository.UserHistoryQuery{
			UserID:           in.UserID,
			CharacterID:      in.CharacterID,
			Variant:          in.Variant,
			ExcludeSessionID: in.SessionID,
			Limit:            pastSessionLimit,
		})
		if err != nil {
			return ChatResult{}, err
		}
		history = append(past, history...)
	}

	userAt := s.now()
	sent := false
	trackDelta := func(d string) error {
		sent = true
		return onDelta(d)
	}
	reply, err := s.llm.Stream(ctx, llm.Request{
		Model:        modelID,
		SystemPrompt: systemPrompt,
		History:      history,
		Message:      in.Message,
	}, trackDelta)

	// チューニング版が Google 側の上限に達したら、プロンプト版のモデルで作り直す。
	// 返答を途中まで送っていたら、二重にならないよう作り直さない
	fallback := false
	if err != nil && in.Variant == model.VariantTuned && !sent && errors.Is(err, llm.ErrQuotaExceeded) {
		log.Printf("chat: チューニング版が上限に達したのでプロンプト版で返答します: %v", err)
		fallback = true
		reply, err = s.llm.Stream(ctx, llm.Request{
			Model:        c.BaseModel,
			SystemPrompt: c.SystemPrompt,
			History:      history,
			Message:      in.Message,
		}, onDelta)
	}
	if err != nil {
		return ChatResult{}, err
	}

	var overlap *model.Overlap
	if s.memo != nil {
		if o, ok := s.memo.Overlap(reply); ok {
			overlap = &o
		}
	}

	base := model.Message{UserID: in.UserID, SessionID: in.SessionID, CharacterID: in.CharacterID, Variant: in.Variant}
	user, bot := base, base
	user.Role, user.Content, user.CreatedAt = "user", in.Message, userAt
	bot.Role, bot.Content, bot.CreatedAt, bot.Overlap, bot.Fallback = "model", reply, s.now(), overlap, fallback
	// 返答の送信後に保存するので、クライアントが切断していても保存できるよう ctx を切り離す
	if err := s.messages.Append(context.WithoutCancel(ctx), user, bot); err != nil {
		return ChatResult{}, err
	}

	return ChatResult{Reply: reply, Overlap: overlap, Fallback: fallback}, nil
}
