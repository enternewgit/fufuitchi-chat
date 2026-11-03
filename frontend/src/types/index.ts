// Go 側の internal/model と形をそろえる
export type Character = {
  id: string;
  name: string;
  systemPrompt?: string; // 管理画面でのみ返る
  baseModel?: string;
  tunedModel?: string;
};

export type Message = {
  sessionId: string;
  characterId: string;
  role: "user" | "model";
  content: string;
  createdAt: string;
};
