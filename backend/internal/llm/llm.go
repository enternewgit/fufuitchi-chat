// Package llm は Gemini の呼び出しをまとめる。
// ベースモデルとチューニング済みモデルの違いは「モデルID」だけにする。
package llm

// TODO:
//   - google.golang.org/genai を使ったクライアントの初期化（Vertex AI バックエンド）
//   - システムプロンプト + 会話履歴 + 新しいメッセージを渡して、返答をストリーミングで受け取る関数
