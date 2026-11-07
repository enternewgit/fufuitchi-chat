package auth

import (
	"context"
	"testing"
	"time"
)

func TestAllowlist(t *testing.T) {
	a := ParseAllowlist(" Admin@Example.com , other@example.com,,")
	if len(a) != 2 {
		t.Fatalf("len = %d", len(a))
	}
	if !a.Allowed("admin@example.com") || !a.Allowed("ADMIN@example.com ") {
		t.Error("大文字小文字や空白は無視して一致するはず")
	}
	if a.Allowed("evil@example.com") || a.Allowed("") {
		t.Error("一覧にないアドレスは許可しないはず")
	}
	if ParseAllowlist("").Allowed("") {
		t.Error("空の一覧は誰も許可しないはず")
	}
}

func TestSessions(t *testing.T) {
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	s := NewSessions(time.Hour)
	s.now = func() time.Time { return now }

	token, sess, err := s.Create(context.Background(), "admin@example.com")
	if err != nil || len(token) < 40 || sess.Email != "admin@example.com" {
		t.Fatalf("Create = %q, %+v, %v", token, sess, err)
	}
	if other, _, _ := s.Create(context.Background(), "admin@example.com"); other == token {
		t.Fatal("トークンは毎回ちがうはず")
	}
	if got, ok := s.Lookup(context.Background(), token); !ok || got.Email != "admin@example.com" {
		t.Fatalf("Lookup = %+v, %v", got, ok)
	}

	now = now.Add(time.Hour)
	if _, ok := s.Lookup(context.Background(), token); ok {
		t.Fatal("期限切れのセッションは無効のはず")
	}

	token, _, _ = s.Create(context.Background(), "admin@example.com")
	s.Delete(context.Background(), token)
	if _, ok := s.Lookup(context.Background(), token); ok {
		t.Fatal("削除したセッションは無効のはず")
	}
}

func TestGoogleVerifierRequiresClientID(t *testing.T) {
	if _, err := (GoogleVerifier{}).Verify(context.Background(), "x.y.z"); err == nil {
		t.Fatal("クライアントIDがないときは検証を拒否するはず")
	}
}
