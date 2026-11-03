// Package handler は HTTP リクエストを受けて返すだけの層。
// ロジックを書き始めたら service に移すサイン。
package handler

// TODO: POST /api/chat
//   - リクエストの検証（characterId, sessionId, message）
//   - service.ChatService を呼ぶ
//   - 返答を SSE（text/event-stream）で流す
