// Package auth は管理者の Google ログインとセッションを扱う。
//
// ブラウザの「Sign in with Google」が受け取った ID トークンを検証し、
// 許可したメールアドレスならセッションを発行する。
// クライアントシークレットは使わないので、サーバーに秘密を置く必要がない。
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/idtoken"
)

// Identity は ID トークンから分かるログインした人の情報。
type Identity struct {
	Email         string
	EmailVerified bool
}

// Verifier は ID トークンを検証する。テストで差し替えられるように interface にしておく。
type Verifier interface {
	Verify(ctx context.Context, idToken string) (Identity, error)
}

// GoogleVerifier は Google が発行した ID トークンを検証する。
type GoogleVerifier struct {
	ClientID string
}

var googleIssuers = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

func (g GoogleVerifier) Verify(ctx context.Context, token string) (Identity, error) {
	// idtoken.Validate は audience が空だと宛先の確認を飛ばすので、必ず指定する
	if g.ClientID == "" {
		return Identity{}, errors.New("auth: GOOGLE_CLIENT_ID が設定されていません")
	}
	// 署名・宛先（aud）・有効期限は idtoken.Validate が確認する
	p, err := idtoken.Validate(ctx, token, g.ClientID)
	if err != nil {
		return Identity{}, err
	}
	// 発行元（iss）は idtoken.Validate が確認しないので、ここで確認する
	if !googleIssuers[p.Issuer] {
		return Identity{}, errors.New("auth: 発行元が Google ではありません")
	}
	email, _ := p.Claims["email"].(string)
	verified, _ := p.Claims["email_verified"].(bool)
	return Identity{Email: email, EmailVerified: verified}, nil
}

// Allowlist は管理者として許可するメールアドレスの一覧。
type Allowlist map[string]struct{}

// ParseAllowlist はカンマ区切りのメールアドレスを読む。
func ParseAllowlist(csv string) Allowlist {
	a := Allowlist{}
	for _, e := range strings.Split(csv, ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			a[e] = struct{}{}
		}
	}
	return a
}

func (a Allowlist) Allowed(email string) bool {
	_, ok := a[strings.ToLower(strings.TrimSpace(email))]
	return ok
}

// Session はログイン中の管理者。
type Session struct {
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Sessions はセッションをメモリに保存する。backend を再起動すると全員ログアウトになる。
type Sessions struct {
	mu  sync.Mutex
	m   map[string]Session
	ttl time.Duration
	now func() time.Time
}

func NewSessions(ttl time.Duration) *Sessions {
	return &Sessions{m: map[string]Session{}, ttl: ttl, now: time.Now}
}

// Create は新しいセッションを作り、そのトークンを返す。
func (s *Sessions) Create(email string) (string, Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	sess := Session{Email: email, ExpiresAt: s.now().Add(s.ttl)}

	s.mu.Lock()
	defer s.mu.Unlock()
	// 期限切れのセッションをついでに掃除する
	for t, old := range s.m {
		if !s.now().Before(old.ExpiresAt) {
			delete(s.m, t)
		}
	}
	s.m[token] = sess
	return token, sess, nil
}

// Lookup は有効なセッションを返す。期限切れなら削除して ok=false。
func (s *Sessions) Lookup(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[token]
	if !ok {
		return Session{}, false
	}
	if !s.now().Before(sess.ExpiresAt) {
		delete(s.m, token)
		return Session{}, false
	}
	return sess, true
}

func (s *Sessions) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, token)
}

type ctxKey struct{}

// WithSession はリクエストの context にログイン中の管理者を入れる。
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

// FromContext はログイン中の管理者を取り出す。
func FromContext(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(ctxKey{}).(Session)
	return s, ok
}
