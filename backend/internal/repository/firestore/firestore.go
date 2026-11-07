// Package firestore は Firestore に保存する実装。backend を再起動してもデータが残る。
//
// コレクション:
//   - characters/{キャラID}        キャラ設定
//   - messages/{自動ID}            会話の1発言
//   - sessions/{セッションID}       会話セッションの集計（管理画面の一覧用）
//   - adminSessions/{トークンのハッシュ} 管理者のログインセッション
//
// 複合インデックスが要らないように、検索は等号の条件だけにして、並べ替えは Go 側で行う。
// 1セッションの発言数は多くないので、これで十分。
package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/auth"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
)

const (
	colCharacters    = "characters"
	colMessages      = "messages"
	colSessions      = "sessions"
	colAdminSessions = "adminSessions"
)

func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

// ---- キャラ設定 ----

type characterDoc struct {
	Name              string `firestore:"name"`
	Description       string `firestore:"description"`
	SystemPrompt      string `firestore:"systemPrompt"`
	BaseModel         string `firestore:"baseModel"`
	TunedModel        string `firestore:"tunedModel"`
	TunedSystemPrompt string `firestore:"tunedSystemPrompt"`
}

func toCharacterDoc(c model.Character) characterDoc {
	return characterDoc{c.Name, c.Description, c.SystemPrompt, c.BaseModel, c.TunedModel, c.TunedSystemPrompt}
}

func (d characterDoc) model(id string) model.Character {
	return model.Character{
		ID: id, Name: d.Name, Description: d.Description, SystemPrompt: d.SystemPrompt,
		BaseModel: d.BaseModel, TunedModel: d.TunedModel, TunedSystemPrompt: d.TunedSystemPrompt,
	}
}

// CharacterRepo は repository.CharacterRepository の Firestore 実装。
type CharacterRepo struct {
	c *firestore.Client
}

func NewCharacterRepo(c *firestore.Client) *CharacterRepo {
	return &CharacterRepo{c: c}
}

// Seed は、まだ Firestore にないキャラだけを追加する。
// 管理画面で編集した内容を、起動のたびに data/characters/ の内容で上書きしないため。
func (r *CharacterRepo) Seed(ctx context.Context, cs []model.Character) (int, error) {
	added := 0
	for _, c := range cs {
		_, err := r.c.Collection(colCharacters).Doc(c.ID).Create(ctx, toCharacterDoc(c))
		if status.Code(err) == codes.AlreadyExists {
			continue
		}
		if err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

func (r *CharacterRepo) List(ctx context.Context) ([]model.Character, error) {
	docs, err := r.c.Collection(colCharacters).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]model.Character, 0, len(docs))
	for _, d := range docs {
		var cd characterDoc
		if err := d.DataTo(&cd); err != nil {
			return nil, err
		}
		out = append(out, cd.model(d.Ref.ID))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *CharacterRepo) Get(ctx context.Context, id string) (model.Character, error) {
	d, err := r.c.Collection(colCharacters).Doc(id).Get(ctx)
	if isNotFound(err) {
		return model.Character{}, repository.ErrNotFound
	}
	if err != nil {
		return model.Character{}, err
	}
	var cd characterDoc
	if err := d.DataTo(&cd); err != nil {
		return model.Character{}, err
	}
	return cd.model(id), nil
}

func (r *CharacterRepo) Save(ctx context.Context, c model.Character) error {
	_, err := r.c.Collection(colCharacters).Doc(c.ID).Set(ctx, toCharacterDoc(c))
	return err
}

func (r *CharacterRepo) Delete(ctx context.Context, id string) error {
	ref := r.c.Collection(colCharacters).Doc(id)
	if _, err := ref.Get(ctx); isNotFound(err) {
		return repository.ErrNotFound
	} else if err != nil {
		return err
	}
	_, err := ref.Delete(ctx)
	return err
}

// ---- 会話ログ ----

type overlapDoc struct {
	Longest int  `firestore:"longest"`
	Exact   bool `firestore:"exact"`
}

type messageDoc struct {
	UserID      string      `firestore:"userId,omitempty"`
	SessionID   string      `firestore:"sessionId"`
	CharacterID string      `firestore:"characterId"`
	Variant     string      `firestore:"variant"`
	Role        string      `firestore:"role"`
	Content     string      `firestore:"content"`
	CreatedAt   time.Time   `firestore:"createdAt"`
	Overlap     *overlapDoc `firestore:"overlap,omitempty"`
	Fallback    bool        `firestore:"fallback,omitempty"`
}

func toMessageDoc(m model.Message) messageDoc {
	d := messageDoc{m.UserID, m.SessionID, m.CharacterID, string(m.Variant), m.Role, m.Content, m.CreatedAt, nil, m.Fallback}
	if m.Overlap != nil {
		d.Overlap = &overlapDoc{m.Overlap.Longest, m.Overlap.Exact}
	}
	return d
}

func (d messageDoc) model() model.Message {
	m := model.Message{
		UserID: d.UserID, SessionID: d.SessionID, CharacterID: d.CharacterID, Variant: model.Variant(d.Variant),
		Role: d.Role, Content: d.Content, CreatedAt: d.CreatedAt, Fallback: d.Fallback,
	}
	if d.Overlap != nil {
		m.Overlap = &model.Overlap{Longest: d.Overlap.Longest, Exact: d.Overlap.Exact}
	}
	return m
}

type sessionDoc struct {
	CharacterID  string    `firestore:"characterId"`
	MessageCount int       `firestore:"messageCount"`
	FirstAt      time.Time `firestore:"firstAt"`
	LastAt       time.Time `firestore:"lastAt"`
}

// MessageRepo は repository.MessageRepository の Firestore 実装。
type MessageRepo struct {
	c *firestore.Client
}

func NewMessageRepo(c *firestore.Client) *MessageRepo {
	return &MessageRepo{c: c}
}

// Append は発言を保存し、同じトランザクションでセッションの集計も更新する。
func (r *MessageRepo) Append(ctx context.Context, msgs ...model.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	// 1回の呼び出しの発言は同じセッションのもの（ChatService はユーザー発言と返答を一緒に保存する）
	sessRef := r.c.Collection(colSessions).Doc(msgs[0].SessionID)
	return r.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var sd sessionDoc
		snap, err := tx.Get(sessRef)
		switch {
		case isNotFound(err):
			sd = sessionDoc{CharacterID: msgs[0].CharacterID, FirstAt: msgs[0].CreatedAt}
		case err != nil:
			return err
		default:
			if err := snap.DataTo(&sd); err != nil {
				return err
			}
		}
		for _, m := range msgs {
			if m.SessionID != msgs[0].SessionID {
				return errors.New("firestore: Append には同じセッションの発言だけを渡してください")
			}
			if err := tx.Create(r.c.Collection(colMessages).NewDoc(), toMessageDoc(m)); err != nil {
				return err
			}
			sd.MessageCount++
			if m.CreatedAt.After(sd.LastAt) {
				sd.LastAt = m.CreatedAt
			}
		}
		return tx.Set(sessRef, sd)
	})
}

// sessionMessages はセッションの発言を、追加された順（createdAt 順）に返す。
func (r *MessageRepo) sessionMessages(ctx context.Context, q firestore.Query) ([]model.Message, error) {
	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]model.Message, 0, len(docs))
	for _, d := range docs {
		var md messageDoc
		if err := d.DataTo(&md); err != nil {
			return nil, err
		}
		out = append(out, md.model())
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *MessageRepo) History(ctx context.Context, q repository.HistoryQuery) ([]model.Message, error) {
	out, err := r.sessionMessages(ctx, r.c.Collection(colMessages).
		Where("sessionId", "==", q.SessionID).
		Where("characterId", "==", q.CharacterID).
		Where("variant", "==", string(q.Variant)))
	if err != nil {
		return nil, err
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[len(out)-q.Limit:]
	}
	return out, nil
}

func (r *MessageRepo) UserHistory(ctx context.Context, q repository.UserHistoryQuery) ([]model.Message, error) {
	all, err := r.sessionMessages(ctx, r.c.Collection(colMessages).
		Where("userId", "==", q.UserID).
		Where("characterId", "==", q.CharacterID).
		Where("variant", "==", string(q.Variant)))
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, m := range all {
		if m.SessionID != q.ExcludeSessionID {
			out = append(out, m)
		}
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[len(out)-q.Limit:]
	}
	return out, nil
}

func (r *MessageRepo) SessionMessages(ctx context.Context, sessionID string) ([]model.Message, error) {
	return r.sessionMessages(ctx, r.c.Collection(colMessages).Where("sessionId", "==", sessionID))
}

func (r *MessageRepo) ListSessions(ctx context.Context, f repository.SessionFilter) ([]model.SessionSummary, error) {
	q := r.c.Collection(colSessions).Query
	if f.CharacterID != "" {
		q = q.Where("characterId", "==", f.CharacterID)
	}
	it := q.Documents(ctx)
	defer it.Stop()
	var out []model.SessionSummary
	for {
		d, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var sd sessionDoc
		if err := d.DataTo(&sd); err != nil {
			return nil, err
		}
		if !f.From.IsZero() && sd.LastAt.Before(f.From) {
			continue
		}
		if !f.To.IsZero() && !sd.FirstAt.Before(f.To) {
			continue
		}
		out = append(out, model.SessionSummary{
			SessionID: d.Ref.ID, CharacterID: sd.CharacterID, MessageCount: sd.MessageCount,
			FirstAt: sd.FirstAt, LastAt: sd.LastAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastAt.After(out[j].LastAt) })
	return out, nil
}

// ---- 管理者のログインセッション ----

type adminSessionDoc struct {
	Email     string    `firestore:"email"`
	ExpiresAt time.Time `firestore:"expiresAt"`
}

// SessionStore は auth.SessionStore の Firestore 実装。
// トークンそのものは保存せず、SHA-256 のハッシュをドキュメントIDにする（DB が漏れてもなりすませない）。
type SessionStore struct {
	c   *firestore.Client
	ttl time.Duration
	now func() time.Time
}

func NewSessionStore(c *firestore.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{c: c, ttl: ttl, now: time.Now}
}

func tokenKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *SessionStore) Create(ctx context.Context, email string) (string, auth.Session, error) {
	token, err := auth.NewToken()
	if err != nil {
		return "", auth.Session{}, err
	}
	sess := auth.Session{Email: email, ExpiresAt: s.now().Add(s.ttl)}
	_, err = s.c.Collection(colAdminSessions).Doc(tokenKey(token)).Create(ctx, adminSessionDoc{sess.Email, sess.ExpiresAt})
	if err != nil {
		return "", auth.Session{}, err
	}
	return token, sess, nil
}

func (s *SessionStore) Lookup(ctx context.Context, token string) (auth.Session, bool) {
	ref := s.c.Collection(colAdminSessions).Doc(tokenKey(token))
	d, err := ref.Get(ctx)
	if isNotFound(err) {
		return auth.Session{}, false
	}
	if err != nil {
		log.Printf("firestore: lookup admin session: %v", err)
		return auth.Session{}, false
	}
	var sd adminSessionDoc
	if err := d.DataTo(&sd); err != nil {
		log.Printf("firestore: decode admin session: %v", err)
		return auth.Session{}, false
	}
	if !s.now().Before(sd.ExpiresAt) {
		if _, err := ref.Delete(ctx); err != nil {
			log.Printf("firestore: delete expired admin session: %v", err)
		}
		return auth.Session{}, false
	}
	return auth.Session{Email: sd.Email, ExpiresAt: sd.ExpiresAt}, true
}

func (s *SessionStore) Delete(ctx context.Context, token string) error {
	_, err := s.c.Collection(colAdminSessions).Doc(tokenKey(token)).Delete(ctx)
	return err
}
