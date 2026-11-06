package service

import (
	"context"
	"errors"
	"testing"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/llm"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository/memory"
)

func TestMemorizerScore(t *testing.T) {
	m := &Memorizer{}
	m.Add("絵に描いた餅です！")
	m.Add("お水が一番落ち着きます。")

	if s, ok := m.Score("絵に描いた餅です"); !ok || s != 1 {
		t.Errorf("完全一致: score = %v, %v", s, ok)
	}
	if s, ok := m.Score("今日はいい天気ですね"); !ok || s > 0.2 {
		t.Errorf("無関係: score = %v, %v", s, ok)
	}
	if _, ok := m.Score("はい"); ok {
		t.Error("短すぎる返答は ok=false のはず")
	}
	if _, ok := (&Memorizer{}).Score("絵に描いた餅です"); ok {
		t.Error("参照なしは ok=false のはず")
	}
}

type fakeLLM struct {
	got   llm.Request
	reply []string
}

func (f *fakeLLM) Stream(ctx context.Context, req llm.Request, onDelta func(string) error) (string, error) {
	f.got = req
	full := ""
	for _, d := range f.reply {
		full += d
		if err := onDelta(d); err != nil {
			return full, err
		}
	}
	return full, nil
}

func TestChatService(t *testing.T) {
	ctx := context.Background()
	chars := memory.NewCharacterRepo()
	chars.Save(ctx, model.Character{ID: "c", SystemPrompt: "sys", BaseModel: "base-model"})
	msgs := memory.NewMessageRepo()
	memo := &Memorizer{}
	memo.Add("絵に描いた餅です！")
	gen := &fakeLLM{reply: []string{"絵に描いた", "餅です！"}}
	svc := NewChatService(chars, msgs, gen, memo)

	in := ChatInput{SessionID: "s", CharacterID: "c", Variant: model.VariantBase, Message: "好きな食べ物は？"}
	var deltas []string
	res, err := svc.Chat(ctx, in, func(d string) error { deltas = append(deltas, d); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if res.Reply != "絵に描いた餅です！" || len(deltas) != 2 {
		t.Fatalf("reply = %q, deltas = %v", res.Reply, deltas)
	}
	if res.Memorization == nil || *res.Memorization != 1 {
		t.Fatalf("memorization = %v", res.Memorization)
	}
	if gen.got.Model != "base-model" || gen.got.SystemPrompt != "sys" {
		t.Fatalf("llm request = %+v", gen.got)
	}

	// 2回目は1往復分の履歴が渡る
	svc.Chat(ctx, in, func(string) error { return nil })
	if len(gen.got.History) != 2 || gen.got.History[0].Role != "user" || gen.got.History[1].Role != "model" {
		t.Fatalf("history = %+v", gen.got.History)
	}
	saved, _ := msgs.History(ctx, repository.HistoryQuery{SessionID: "s", CharacterID: "c", Variant: model.VariantBase})
	if len(saved) != 4 {
		t.Fatalf("saved = %d messages", len(saved))
	}
}

func TestChatServiceErrors(t *testing.T) {
	ctx := context.Background()
	chars := memory.NewCharacterRepo()
	chars.Save(ctx, model.Character{ID: "c", BaseModel: "m"})
	svc := NewChatService(chars, memory.NewMessageRepo(), &fakeLLM{}, nil)
	noop := func(string) error { return nil }

	_, err := svc.Chat(ctx, ChatInput{CharacterID: "none", Variant: model.VariantBase}, noop)
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("存在しないキャラ: %v", err)
	}
	_, err = svc.Chat(ctx, ChatInput{CharacterID: "c", Variant: model.VariantTuned}, noop)
	if !errors.Is(err, ErrNoTunedModel) {
		t.Errorf("チューニング未作成: %v", err)
	}
}
