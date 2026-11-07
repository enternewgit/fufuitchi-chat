"use client";

import { useEffect, useRef, useState } from "react";
import { fetchCharacter, sendMessage } from "@/lib/api";
import { getSessionId, resetSessionId } from "@/lib/session";
import type { PublicCharacter, Variant } from "@/types";
import ChatInput from "./ChatInput";
import MessageList, { type ChatMessage } from "./MessageList";

export default function ChatRoom({ characterId }: { characterId: string }) {
  const [character, setCharacter] = useState<PublicCharacter | null>(null);
  const [loadError, setLoadError] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [sending, setSending] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    fetchCharacter(characterId)
      .then(setCharacter)
      .catch((e: Error) => setLoadError(e.message));
    return () => abortRef.current?.abort();
  }, [characterId]);

  // 最後のメッセージ（返答中のもの）を書き換える
  const updateLast = (fn: (m: ChatMessage) => ChatMessage) =>
    setMessages((prev) => {
      const last = prev[prev.length - 1];
      // 「新しい会話」で消えた後に届いた返答は捨てる
      if (!last?.pending) return prev;
      return [...prev.slice(0, -1), fn(last)];
    });

  const send = async (text: string) => {
    if (!character) return;
    // チューニング版があればそれを使う（上限に達したら backend がプロンプト版で返答する）
    const variant: Variant = character.hasTuned ? "tuned" : "base";
    const abort = new AbortController();
    abortRef.current = abort;
    setSending(true);
    setMessages((prev) => [
      ...prev,
      { role: "user", content: text },
      { role: "model", content: "", pending: true },
    ]);

    try {
      await sendMessage(
        { characterId, sessionId: getSessionId(), variant, message: text },
        (delta) => updateLast((m) => ({ ...m, content: m.content + delta })),
        abort.signal,
      );
      updateLast((m) => ({ ...m, pending: false }));
    } catch (e) {
      if (!abort.signal.aborted) {
        updateLast((m) => ({ ...m, pending: false, error: (e as Error).message }));
      }
    }
    setSending(false);
  };

  const newConversation = () => {
    abortRef.current?.abort();
    resetSessionId();
    setMessages([]);
    setSending(false);
  };

  if (loadError) {
    return <main className="p-4 text-red-600">キャラを読み込めませんでした（{loadError}）</main>;
  }
  if (!character) {
    return <main className="p-4 text-muted">読み込み中…</main>;
  }

  return (
    <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col px-4 py-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-xl font-bold">{character.name}</h1>
        <button
          type="button"
          onClick={newConversation}
          className="rounded border border-line px-3 py-1 text-sm hover:bg-surface"
        >
          新しい会話
        </button>
      </div>

      <section className="mt-4 flex min-h-[50vh] flex-1 flex-col rounded-lg border border-line">
        <MessageList messages={messages} characterName={character.name} />
      </section>

      <ChatInput onSend={send} disabled={sending} />
      <p className="mt-2 text-xs text-muted">
        会話の内容は記録され、サービスの改善のために管理者が閲覧することがあります。
      </p>
    </main>
  );
}
