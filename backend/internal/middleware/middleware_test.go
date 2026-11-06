package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/auth"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func TestAdminAuth(t *testing.T) {
	sessions := auth.NewSessions(time.Hour)
	token, _, err := sessions.Create("admin@example.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		header string
		want   int
	}{
		{"Bearer " + token, http.StatusOK},
		{"Bearer wrong", http.StatusUnauthorized},
		{token, http.StatusUnauthorized},
		{"", http.StatusUnauthorized},
		{"Bearer ", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		var got auth.Session
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, _ = auth.FromContext(r.Context())
		})
		req := httptest.NewRequest("GET", "/api/admin/characters", nil)
		req.Header.Set("Authorization", tt.header)
		rec := httptest.NewRecorder()
		AdminAuth(sessions)(next).ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("header=%q: got %d, want %d", tt.header, rec.Code, tt.want)
		}
		if tt.want == http.StatusOK && got.Email != "admin@example.com" {
			t.Errorf("context のセッション = %+v", got)
		}
	}
}

func TestCORS(t *testing.T) {
	h := CORS("http://localhost:3000")(ok)

	req := httptest.NewRequest("OPTIONS", "/api/chat", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("許可したオリジン: %d %v", rec.Code, rec.Header())
	}

	req = httptest.NewRequest("GET", "/api/characters", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("許可していないオリジンにヘッダーを付けてはいけない")
	}
}

func TestLimiter(t *testing.T) {
	l := &limiter{burst: 2, every: time.Second, buckets: map[string]*bucket{}}
	now := time.Now()
	if !l.allow("a", now) || !l.allow("a", now) {
		t.Fatal("burst 分は通るはず")
	}
	if l.allow("a", now) {
		t.Fatal("burst を超えたら拒否するはず")
	}
	if !l.allow("b", now) {
		t.Fatal("別のクライアントは通るはず")
	}
	if !l.allow("a", now.Add(time.Second)) {
		t.Fatal("時間がたてば回復するはず")
	}
}
