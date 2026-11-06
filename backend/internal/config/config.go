package config

import "os"

// Config はアプリ全体の設定。値はすべて環境変数から読む。
type Config struct {
	Port           string
	GeminiAPIKey   string // 設定されていれば Gemini API、空なら Vertex AI を使う
	ProjectID      string
	Location       string
	GoogleClientID string // 管理者ログイン用（公開してよい値）
	AdminEmails    string // 管理者として許可するメールアドレス（カンマ区切り）
	AllowedOrigin  string
	DataDir        string // characters/ と processed/ を含むディレクトリ
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8080"),
		GeminiAPIKey:   os.Getenv("GEMINI_API_KEY"),
		ProjectID:      os.Getenv("GOOGLE_CLOUD_PROJECT"),
		Location:       getEnv("GOOGLE_CLOUD_LOCATION", "us-central1"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		AdminEmails:    os.Getenv("ADMIN_EMAILS"),
		AllowedOrigin:  getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		DataDir:        getEnv("DATA_DIR", "../data"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
