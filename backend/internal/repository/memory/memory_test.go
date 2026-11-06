package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
)

func TestCharacterRepo(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"id":"a","name":"A","baseModel":"m"}`), 0o644)

	r := NewCharacterRepo()
	if n, err := r.LoadDir(dir); err != nil || n != 1 {
		t.Fatalf("LoadDir = %d, %v", n, err)
	}
	if c, err := r.Get(ctx, "a"); err != nil || c.Name != "A" {
		t.Fatalf("Get = %+v, %v", c, err)
	}
	r.Save(ctx, model.Character{ID: "b", Name: "B"})
	if cs, _ := r.List(ctx); len(cs) != 2 || cs[0].ID != "a" || cs[1].ID != "b" {
		t.Fatalf("List = %+v", cs)
	}
	if err := r.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(ctx, "a"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Get after delete: %v", err)
	}
}

func TestMessageRepoHistory(t *testing.T) {
	ctx := context.Background()
	r := NewMessageRepo()
	msg := func(session string, v model.Variant, content string) model.Message {
		return model.Message{SessionID: session, CharacterID: "c", Variant: v, Role: "user", Content: content}
	}
	r.Append(ctx,
		msg("s1", model.VariantBase, "1"),
		msg("s1", model.VariantTuned, "x"),
		msg("s2", model.VariantBase, "y"),
		msg("s1", model.VariantBase, "2"),
		msg("s1", model.VariantBase, "3"),
	)

	got, _ := r.History(ctx, repository.HistoryQuery{SessionID: "s1", CharacterID: "c", Variant: model.VariantBase, Limit: 2})
	if len(got) != 2 || got[0].Content != "2" || got[1].Content != "3" {
		t.Fatalf("History = %+v", got)
	}
}

func TestMessageRepoListSessions(t *testing.T) {
	ctx := context.Background()
	r := NewMessageRepo()
	day := func(d int) time.Time { return time.Date(2026, 11, d, 12, 0, 0, 0, time.UTC) }
	r.Append(ctx,
		model.Message{SessionID: "old", CharacterID: "c", CreatedAt: day(1)},
		model.Message{SessionID: "new", CharacterID: "c", CreatedAt: day(5)},
		model.Message{SessionID: "new", CharacterID: "c", CreatedAt: day(6)},
		model.Message{SessionID: "other", CharacterID: "d", CreatedAt: day(6)},
	)

	got, _ := r.ListSessions(ctx, repository.SessionFilter{CharacterID: "c", From: day(3)})
	if len(got) != 1 || got[0].SessionID != "new" || got[0].MessageCount != 2 {
		t.Fatalf("ListSessions = %+v", got)
	}
	got, _ = r.ListSessions(ctx, repository.SessionFilter{To: day(2)})
	if len(got) != 1 || got[0].SessionID != "old" {
		t.Fatalf("ListSessions(To) = %+v", got)
	}
}
