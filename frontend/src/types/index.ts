// Go 側の internal/model と形をそろえる

// 一般ユーザーに返るキャラ情報（GET /api/characters）
export type PublicCharacter = {
  id: string;
  name: string;
  description: string;
  hasTuned: boolean;
};

// 管理画面でのみ返るキャラ設定（GET /api/admin/characters）
export type Character = {
  id: string;
  name: string;
  description: string;
  systemPrompt: string;
  baseModel: string;
  tunedModel: string;
};

// base: プロンプト版 / tuned: チューニング版
export type Variant = "base" | "tuned";

export type Message = {
  sessionId: string;
  characterId: string;
  variant: Variant;
  role: "user" | "model";
  content: string;
  createdAt: string;
  memorization?: number;
};

export type SessionSummary = {
  sessionId: string;
  characterId: string;
  messageCount: number;
  firstAt: string;
  lastAt: string;
};

// POST /api/chat が SSE で送ってくるイベント
export type ChatEvent =
  | { type: "delta"; text: string }
  | { type: "done"; memorization?: number }
  | { type: "error"; error: string };
