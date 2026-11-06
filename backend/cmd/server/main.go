package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata" // distroless にはタイムゾーンのデータがないので埋め込む

	"github.com/enternewgit/fufuitchi-chat/backend/internal/auth"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/config"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/handler"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/llm"
	"github.com/enternewgit/fufuitchi-chat/backend/internal/middleware"
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

	chars := memory.NewCharacterRepo()
	n, err := chars.LoadDir(filepath.Join(cfg.DataDir, "characters"))
	if err != nil {
		log.Fatalf("load characters: %v", err)
	}
	log.Printf("loaded %d characters", n)
	messages := memory.NewMessageRepo()

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
	sessions := auth.NewSessions(7 * 24 * time.Hour)
	authH := handler.NewAuthHandler(auth.GoogleVerifier{ClientID: cfg.GoogleClientID}, allow, sessions)

	mux := http.NewServeMux()

	// 動作確認用
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /api/characters", charH.List)
	mux.HandleFunc("GET /api/characters/{id}", charH.Get)
	// 比較モードでは1回の送信で2リクエスト来るので、20回分まで連続で受け付ける
	mux.Handle("POST /api/chat", middleware.RateLimit(20, 3*time.Second)(http.HandlerFunc(chatH.Post)))

	mux.Handle("POST /api/auth/google", middleware.RateLimit(10, 6*time.Second)(http.HandlerFunc(authH.Google)))
	mux.HandleFunc("POST /api/auth/logout", authH.Logout)

	admin := http.NewServeMux()
	admin.HandleFunc("GET /api/admin/me", authH.Me)
	admin.HandleFunc("GET /api/admin/characters", adminH.ListCharacters)
	admin.HandleFunc("POST /api/admin/characters", adminH.CreateCharacter)
	admin.HandleFunc("PUT /api/admin/characters/{id}", adminH.UpdateCharacter)
	admin.HandleFunc("DELETE /api/admin/characters/{id}", adminH.DeleteCharacter)
	admin.HandleFunc("GET /api/admin/logs", adminH.ListLogs)
	admin.HandleFunc("GET /api/admin/logs/{sessionId}", adminH.GetLog)
	mux.Handle("/api/admin/", middleware.AdminAuth(sessions)(admin))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           middleware.Logging(middleware.CORS(cfg.AllowedOrigin)(mux)),
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
