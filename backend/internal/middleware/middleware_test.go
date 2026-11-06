package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func TestAdminAuth(t *testing.T) {
	tests := []struct {
		token, header string
		want          int
	}{
		{"secret", "Bearer secret", http.StatusOK},
		{"secret", "Bearer wrong", http.StatusUnauthorized},
		{"secret", "secret", http.StatusUnauthorized},
		{"secret", "", http.StatusUnauthorized},
		{"", "Bearer ", http.StatusUnauthorized}, // ADMIN_TOKEN 未設定ならすべて拒否
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/api/admin/characters", nil)
		req.Header.Set("Authorization", tt.header)
		rec := httptest.NewRecorder()
		AdminAuth(tt.token)(ok).ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("token=%q header=%q: got %d, want %d", tt.token, tt.header, rec.Code, tt.want)
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
