package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/llm"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository/memory"
)

func TestMemorizerOverlap(t *testing.T) {
	m := &Memorizer{}
	m.Add("絵に描いた餅です！")
	m.Add("お水が一番落ち着きます。")

	tests := []struct {
		output  string
		longest int
		exact   bool
	}{
		{"絵に描いた餅です", 8, true},      // 記号を除くと丸ごと同じ
		{"好物は絵に描いた餅ですよ", 8, false}, // 一部が一致
		{"今日はいい天気ですね", 2, false},   // 「です」程度しか一致しない
		{"はい", 1, false},           // 短い返答でも値が膨らまない（「い」の1文字だけ）
	}
	for _, tt := range tests {
		o, ok := m.Overlap(tt.output)
		if !ok || o.Longest != tt.longest || o.Exact != tt.exact {
			t.Errorf("%q: got %+v, %v; want longest=%d exact=%v", tt.output, o, ok, tt.longest, tt.exact)
		}
	}
	if _, ok := (&Memorizer{}).Overlap("絵に描いた餅です"); ok {
		t.Error("参照なしは ok=false のはず")
	}
}

type fakeLLM struct {
	got   llm.Request
	reply []string
	// errFor のモデルを呼んだら、返答を送る前にこのエラーを返す
	errFor string
	err    error
}

func (f *fakeLLM) Stream(ctx context.Context, req llm.Request, onDelta func(string) error) (string, error) {
	f.got = req
	if f.errFor != "" && req.Model == f.errFor {
		return "", f.err
	}
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
	if res.Overlap == nil || !res.Overlap.Exact {
		t.Fatalf("overlap = %+v", res.Overlap)
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

func TestChatServiceTuned(t *testing.T) {
	ctx := context.Background()
	chars := memory.NewCharacterRepo()
	chars.Save(ctx, model.Character{ID: "c", SystemPrompt: "sys", BaseModel: "base-model", TunedModel: "tuned-model"})
	gen := &fakeLLM{reply: []string{"はい"}}
	svc := NewChatService(chars, memory.NewMessageRepo(), gen, nil)

	in := ChatInput{SessionID: "s", CharacterID: "c", Variant: model.VariantTuned, Message: "こんにちは"}
	for range 5 {
		if _, err := svc.Chat(ctx, in, func(string) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if gen.got.Model != "tuned-model" {
		t.Errorf("model = %q", gen.got.Model)
	}
	if gen.got.SystemPrompt != "" {
		t.Errorf("チューニング版には TunedSystemPrompt（空）を渡すはず: %q", gen.got.SystemPrompt)
	}
	// 履歴の上限はないので、それまでの4往復（8発言）が全部渡る
	if len(gen.got.History) != 8 {
		t.Errorf("history = %d, want 8", len(gen.got.History))
	}
}

func TestChatServiceTunedFallback(t *testing.T) {
	ctx := context.Background()
	chars := memory.NewCharacterRepo()
	chars.Save(ctx, model.Character{ID: "c", SystemPrompt: "sys", BaseModel: "base-model", TunedModel: "tuned-model"})
	gen := &fakeLLM{reply: []string{"こんにちは"}, errFor: "tuned-model", err: fmt.Errorf("%w: 429", llm.ErrQuotaExceeded)}
	msgs := memory.NewMessageRepo()
	svc := NewChatService(chars, msgs, gen, nil)

	in := ChatInput{SessionID: "s", CharacterID: "c", Variant: model.VariantTuned, Message: "やあ"}
	res, err := svc.Chat(ctx, in, func(string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !res.Fallback || res.Reply != "こんにちは" {
		t.Fatalf("result = %+v", res)
	}
	if gen.got.Model != "base-model" || gen.got.SystemPrompt != "sys" {
		t.Fatalf("プロンプト版で作り直すはず: %+v", gen.got)
	}
	saved, _ := msgs.History(ctx, repository.HistoryQuery{SessionID: "s", CharacterID: "c", Variant: model.VariantTuned})
	if len(saved) != 2 || !saved[1].Fallback {
		t.Fatalf("saved = %+v", saved)
	}

	// 上限以外のエラーは作り直さない
	gen.err = errors.New("boom")
	if _, err := svc.Chat(ctx, in, func(string) error { return nil }); err == nil {
		t.Fatal("上限以外のエラーはそのまま返すはず")
	}
}

func TestChatServiceRemembersPastSessions(t *testing.T) {
	ctx := context.Background()
	chars := memory.NewCharacterRepo()
	chars.Save(ctx, model.Character{ID: "c", BaseModel: "base-model", TunedModel: "tuned-model"})
	gen := &fakeLLM{reply: []string{"はい"}}
	svc := NewChatService(chars, memory.NewMessageRepo(), gen, nil)
	noop := func(string) error { return nil }

	svc.Chat(ctx, ChatInput{UserID: "u", SessionID: "s1", CharacterID: "c", Variant: model.VariantTuned, Message: "私はたろうです"}, noop)
	// 別の人の会話は混ざらない
	svc.Chat(ctx, ChatInput{UserID: "other", SessionID: "s9", CharacterID: "c", Variant: model.VariantTuned, Message: "私は花子です"}, noop)
	// 新しい会話でも、同じユーザーの前の会話が先頭に入る
	svc.Chat(ctx, ChatInput{UserID: "u", SessionID: "s2", CharacterID: "c", Variant: model.VariantTuned, Message: "私の名前は？"}, noop)

	h := gen.got.History
	if len(h) != 2 || h[0].Content != "私はたろうです" || h[0].SessionID != "s1" {
		t.Fatalf("history = %+v", h)
	}

	// ユーザーIDがなければ引き継がない
	svc.Chat(ctx, ChatInput{SessionID: "s3", CharacterID: "c", Variant: model.VariantTuned, Message: "こんにちは"}, noop)
	if len(gen.got.History) != 0 {
		t.Fatalf("history = %+v", gen.got.History)
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
