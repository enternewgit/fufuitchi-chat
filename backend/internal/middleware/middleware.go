// Package middleware は http.Handler を包む共通処理。
package middleware

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/auth"
)

// AdminAuth は Authorization: Bearer <セッショントークン> が有効なセッションか確認する。
// 有効なら、ログイン中の管理者を context に入れて次に渡す。
func AdminAuth(sessions auth.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			sess, ok := sessions.Lookup(r.Context(), token)
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithSession(r.Context(), sess)))
		})
	}
}

// CORS は allowedOrigin からのリクエストだけ許可する。
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Origin") == allowedOrigin {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", allowedOrigin)
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				h.Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit は同じクライアント（IP）からの連打を防ぐ（Gemini の料金対策）。
// burst 回まで連続で受け付け、その後は every ごとに1回分回復する。
// trustProxy が true なら X-Forwarded-For から接続元を判断する（Cloud Run など、プロキシの後ろで動くとき）。
func RateLimit(burst int, every time.Duration, trustProxy bool) func(http.Handler) http.Handler {
	l := &limiter{burst: float64(burst), every: every, buckets: map[string]*bucket{}}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(clientIP(r, trustProxy), time.Now()) {
				writeError(w, http.StatusTooManyRequests, "リクエストが多すぎます。少し待ってからもう一度送ってください")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type bucket struct {
	tokens float64
	last   time.Time
}

type limiter struct {
	mu      sync.Mutex
	burst   float64
	every   time.Duration
	buckets map[string]*bucket
}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += float64(now.Sub(b.last)) / float64(l.every)
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		// プロキシは自分が見た接続元を末尾に足すので、末尾が本当の接続元。
		// 先頭のほうはクライアントが自由に書けるので信用しない
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Logging はリクエストのログを出す。
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap があると http.ResponseController 経由で Flush できる（SSE に必要）。
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
