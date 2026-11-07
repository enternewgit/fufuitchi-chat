"use client";

import { useEffect, useRef } from "react";
import type { Overlap } from "@/types";

export type ChatMessage = {
  role: "user" | "model";
  content: string;
  pending?: boolean; // 返答を受信中
  error?: string;
  overlap?: Overlap; // 学習データとの重なり
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
    <div className="flex-1 space-y-3 overflow-y-auto p-3" aria-live="polite">
      {messages.length === 0 && (
        <p className="text-sm text-muted">{characterName}に話しかけてみましょう。</p>
      )}
      {messages.map((m, i) => (
        <div key={i} className={`flex ${m.role === "user" ? "justify-end" : "justify-start"}`}>
          <div className="max-w-[85%]">
            <div
              className={`whitespace-pre-wrap rounded-2xl px-3 py-2 ${
                m.role === "user" ? "bg-accent text-accent-foreground" : "bg-surface"
              }`}
            >
              {m.content || (m.pending ? "…" : "")}
              {m.error && <span className="text-red-600">{m.error}</span>}
            </div>
            {m.role === "model" && m.overlap && (
              <p
                className="mt-1 text-xs text-muted"
                title="学習データのお返事と、最長で何文字連続して同じか（記号・空白は除く）"
              >
                {formatOverlap(m.overlap)}
              </p>
            )}
          </div>
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  );
}

export function formatOverlap(o: Overlap): string {
  return o.exact ? "学習データと完全一致" : `学習データと最長 ${o.longest} 文字一致`;
}
