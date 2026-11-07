package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/auth"
)

type fakeVerifier map[string]auth.Identity

func (f fakeVerifier) Verify(ctx context.Context, token string) (auth.Identity, error) {
	id, ok := f[token]
	if !ok {
		return auth.Identity{}, errors.New("invalid token")
	}
	return id, nil
}

func TestGoogleLogin(t *testing.T) {
	v := fakeVerifier{
		"admin":      {Email: "admin@example.com", EmailVerified: true},
		"unverified": {Email: "admin@example.com", EmailVerified: false},
		"stranger":   {Email: "stranger@example.com", EmailVerified: true},
	}
	sessions := auth.NewSessions(time.Hour)
	h := NewAuthHandler(v, auth.ParseAllowlist("admin@example.com"), sessions, "memory")

	tests := []struct {
		credential string
		want       int
	}{
		{"admin", http.StatusOK},
		{"unverified", http.StatusForbidden},
		{"stranger", http.StatusForbidden},
		{"forged", http.StatusUnauthorized},
		{"", http.StatusBadRequest},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "/api/auth/google", strings.NewReader(`{"credential":"`+tt.credential+`"}`))
		rec := httptest.NewRecorder()
		h.Google(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s: got %d, want %d", tt.credential, rec.Code, tt.want)
			continue
		}
		if tt.want != http.StatusOK {
			continue
		}
		var res loginResponse
		json.NewDecoder(rec.Body).Decode(&res)
		if _, ok := sessions.Lookup(context.Background(), res.Token); !ok || res.Email != "admin@example.com" {
			t.Errorf("発行したセッションが無効: %+v", res)
		}
	}
}

func TestLogout(t *testing.T) {
	sessions := auth.NewSessions(time.Hour)
	token, _, _ := sessions.Create(context.Background(), "admin@example.com")
	h := NewAuthHandler(fakeVerifier{}, auth.Allowlist{}, sessions, "memory")

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	h.Logout(httptest.NewRecorder(), req)
	if _, ok := sessions.Lookup(context.Background(), token); ok {
		t.Fatal("ログアウト後もセッションが残っている")
	}
}

func TestValidateChatIDs(t *testing.T) {
	ok := chatRequest{CharacterID: "tsukuyomi", SessionID: "0b5c3f2e-1d2a-4c1b-9a8e-3f4d5e6f7a8b", Variant: "tuned", Message: "こんにちは"}
	if msg := validateChat(ok); msg != "" {
		t.Fatalf("正しいリクエストを弾いた: %s", msg)
	}
	bad := []chatRequest{
		{CharacterID: "tsukuyomi", SessionID: "a/b", Variant: "tuned", Message: "x"},
		{CharacterID: "tsukuyomi", SessionID: "s", UserID: "../u", Variant: "tuned", Message: "x"},
		{CharacterID: "a/b", SessionID: "s", Variant: "tuned", Message: "x"},
		{CharacterID: "tsukuyomi", SessionID: strings.Repeat("a", 101), Variant: "tuned", Message: "x"},
	}
	for _, r := range bad {
		if validateChat(r) == "" {
			t.Errorf("不正なIDを通した: %+v", r)
		}
	}
}
