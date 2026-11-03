package config

import "os"

// Config はアプリ全体の設定。値はすべて環境変数から読む。
type Config struct {
	Port          string
	ProjectID     string
	Location      string
	AdminToken    string
	AllowedOrigin string
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		ProjectID:     os.Getenv("GOOGLE_CLOUD_PROJECT"),
		Location:      getEnv("GOOGLE_CLOUD_LOCATION", "us-central1"),
		AdminToken:    os.Getenv("ADMIN_TOKEN"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
