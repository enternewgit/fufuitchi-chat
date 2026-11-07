package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // distroless にはタイムゾーンのデータがないので埋め込む

	"cloud.google.com/go/firestore"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/auth"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/config"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/handler"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/llm"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/middleware"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository"
	fsrepo "github.com/enternewgit/fufuitchi-chat/backend/internal/repository/firestore"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/repository/memory"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	allow := auth.ParseAllowlist(cfg.AdminEmails)
	if cfg.GoogleClientID == "" || len(allow) == 0 {
		log.Print("WARNING: GOOGLE_CLIENT_ID か ADMIN_EMAILS が未設定なので、管理画面にログインできません")
	}

	sessionTTL := 7 * 24 * time.Hour
	chars, messages, sessions, closeStore, err := openStore(ctx, cfg, sessionTTL)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer closeStore()
	if err := applyTunedModelEnv(ctx, chars); err != nil {
		log.Fatalf("tuned model env: %v", err)
	}

	memo, err := service.LoadMemorizer(filepath.Join(cfg.DataDir, "processed"))
	if err != nil {
		log.Fatalf("load memorizer: %v", err)
	}
	log.Printf("memorizer: %d reference replies", memo.Len())

	gen, err := llm.New(ctx, llm.Config{APIKey: cfg.GeminiAPIKey, Project: cfg.ProjectID, Location: cfg.Location})
	if err != nil {
		log.Fatalf("llm: %v", err)
	}

	chatH := handler.NewChatHandler(service.NewChatService(chars, messages, gen, memo))
	charH := handler.NewCharacterHandler(chars)
	adminH := handler.NewAdminHandler(chars, messages)
	authH := handler.NewAuthHandler(auth.GoogleVerifier{ClientID: cfg.GoogleClientID}, allow, sessions, cfg.Storage)

	mux := http.NewServeMux()

	// 動作確認用
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /api/characters", charH.List)
	mux.HandleFunc("GET /api/characters/{id}", charH.Get)
	// 比較モードでは1回の送信で2リクエスト来るので、20回分まで連続で受け付ける
	mux.Handle("POST /api/chat", middleware.RateLimit(20, 3*time.Second, cfg.TrustProxy)(http.HandlerFunc(chatH.Post)))

	mux.Handle("POST /api/auth/google", middleware.RateLimit(10, 6*time.Second, cfg.TrustProxy)(http.HandlerFunc(authH.Google)))
	mux.HandleFunc("POST /api/auth/logout", authH.Logout)

	admin := http.NewServeMux()
	admin.HandleFunc("GET /api/admin/me", authH.Me)
	admin.HandleFunc("GET /api/admin/characters", adminH.ListCharacters)
	admin.HandleFunc("POST /api/admin/characters", adminH.CreateCharacter)
	admin.HandleFunc("PUT /api/admin/characters/{id}", adminH.UpdateCharacter)
	admin.HandleFunc("DELETE /api/admin/characters/{id}", adminH.DeleteCharacter)
	admin.HandleFunc("GET /api/admin/logs", adminH.ListLogs)
	admin.HandleFunc("GET /api/admin/logs/{sessionId}", adminH.GetLog)
	admin.HandleFunc("DELETE /api/admin/logs/{sessionId}", adminH.DeleteLog)
	mux.Handle("/api/admin/", middleware.AdminAuth(sessions)(admin))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           middleware.Logging(middleware.SecurityHeaders(middleware.CORS(cfg.AllowedOrigin)(mux))),
		ReadHeaderTimeout: 10 * time.Second,
		// WriteTimeout は SSE の返答を途中で切ってしまうので設定しない
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// applyTunedModelEnv は、環境変数 TUNED_MODEL_<キャラID> があれば、そのキャラの tunedModel を上書きする。
// チューニング済みモデルのエンドポイントはプロジェクトごとに違うので、
// キャラシート（リポジトリに入れる）ではなく .env に置く。
// キャラIDは大文字にし、- を _ に置き換える（例: tsukuyomi → TUNED_MODEL_TSUKUYOMI）。
func applyTunedModelEnv(ctx context.Context, chars repository.CharacterRepository) error {
	cs, err := chars.List(ctx)
	if err != nil {
		return err
	}
	for _, c := range cs {
		key := "TUNED_MODEL_" + strings.ToUpper(strings.ReplaceAll(c.ID, "-", "_"))
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		c.TunedModel = v
		if err := chars.Save(ctx, c); err != nil {
			return err
		}
		log.Printf("%s: tuned model from %s", c.ID, key)
	}
	return nil
}

// openStore は STORAGE の設定に合わせて保存先を開く。
// キャラ設定は data/characters/ から読み、firestore のときはまだないキャラだけを追加する。
func openStore(ctx context.Context, cfg config.Config, sessionTTL time.Duration) (
	repository.CharacterRepository, repository.MessageRepository, auth.SessionStore, func(), error,
) {
	seed := memory.NewCharacterRepo()
	n, err := seed.LoadDir(filepath.Join(cfg.DataDir, "characters"))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("load characters: %w", err)
	}
	log.Printf("loaded %d characters from files", n)

	switch cfg.Storage {
	case "memory":
		log.Print("storage: memory（再起動するとデータは消えます）")
		return seed, memory.NewMessageRepo(), auth.NewSessions(sessionTTL), func() {}, nil
	case "firestore":
		if cfg.ProjectID == "" {
			return nil, nil, nil, nil, errors.New("STORAGE=firestore には GOOGLE_CLOUD_PROJECT が必要です")
		}
		client, err := firestore.NewClientWithDatabase(ctx, cfg.ProjectID, cfg.FirestoreDatabase)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		chars := fsrepo.NewCharacterRepo(client)
		cs, _ := seed.List(ctx)
		added, err := chars.Seed(ctx, cs)
		if err != nil {
			client.Close()
			return nil, nil, nil, nil, fmt.Errorf("seed characters: %w", err)
		}
		log.Printf("storage: firestore (%s), %d characters added", cfg.FirestoreDatabase, added)
		return chars, fsrepo.NewMessageRepo(client), fsrepo.NewSessionStore(client, sessionTTL), func() { client.Close() }, nil
	}
	return nil, nil, nil, nil, fmt.Errorf("STORAGE は memory か firestore です: %q", cfg.Storage)
}
