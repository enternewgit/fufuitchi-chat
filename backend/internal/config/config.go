package config

import "os"

// Config はアプリ全体の設定。値はすべて環境変数から読む。
type Config struct {
	Port          string
	GeminiAPIKey  string // 設定されていれば Gemini API、空なら Vertex AI を使う
	ProjectID     string
	Location      string
	AdminToken    string
	AllowedOrigin string
	DataDir       string // characters/ と processed/ を含むディレクトリ
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		ProjectID:     os.Getenv("GOOGLE_CLOUD_PROJECT"),
		Location:      getEnv("GOOGLE_CLOUD_LOCATION", "us-central1"),
		AdminToken:    os.Getenv("ADMIN_TOKEN"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		DataDir:       getEnv("DATA_DIR", "../data"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
