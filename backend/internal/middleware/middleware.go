// Package middleware は http.Handler を包む共通処理。
package middleware

// TODO:
//   - AdminAuth: Authorization ヘッダーのトークンを ADMIN_TOKEN と比較（crypto/subtle を使う）
//   - CORS: ALLOWED_ORIGIN だけ許可
//   - RateLimit: 同じクライアントからの連打を防ぐ（Gemini の料金対策）
//   - Logging: リクエストのログ
