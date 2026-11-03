package main

import (
	"log"
	"net/http"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/config"
)

func main() {
	cfg := config.Load()

	mux := http.NewServeMux()

	// 動作確認用
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// TODO: ルーティングを追加する
	//   GET  /api/characters
	//   POST /api/chat
	//   /api/admin/* は middleware.AdminAuth で保護する

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
