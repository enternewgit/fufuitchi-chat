"use client";

import { useEffect, useRef } from "react";

export type ChatMessage = {
  role: "user" | "model";
  content: string;
  pending?: boolean; // 返答を受信中
  error?: string;
};

export default function MessageList({
  messages,
  characterName,
}: {
  messages: ChatMessage[];
  characterName: string;
}) {
  const bottomRef = useRef<HTMLDivElement>(null);

  // 新しいメッセージや返答の続きが届いたら一番下までスクロールする
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: "end" });
  }, [messages]);

  return (
    <div className="flex-1 space-y-3 p-3" aria-live="polite">
      {messages.length === 0 && (
        <p className="text-sm text-muted">{characterName}に話しかけてみましょう。</p>
      )}
      {messages.map((m, i) => (
        <div key={i} className={`flex ${m.role === "user" ? "justify-end" : "justify-start"}`}>
          <div className="max-w-[85%]">
            <div
              className={`whitespace-pre-wrap break-words rounded-2xl px-3 py-2 ${
                m.role === "user" ? "bg-accent text-accent-foreground" : "bg-surface"
              }`}
            >
              {m.content || (m.pending ? "…" : "")}
              {m.error && <span className="text-red-600">{m.error}</span>}
            </div>
          </div>
        </div>
      ))}
      {/* 画面下に固定した入力欄の裏に最後の吹き出しが隠れないよう、その高さ分だけ余白を取って止める */}
      <div ref={bottomRef} className="scroll-mb-28" />
    </div>
  );
}
