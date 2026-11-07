"use client";

import { useEffect, useRef, useState } from "react";
import { fetchCharacter, sendMessage } from "@/lib/api";
import { getSessionId, resetSessionId } from "@/lib/session";
import type { PublicCharacter, Variant } from "@/types";
import ChatInput from "./ChatInput";
import MessageList, { type ChatMessage } from "./MessageList";

// base: プロンプト版だけ / tuned: チューニング版だけ / compare: 両方を並べる
type Mode = Variant | "compare";

const LABELS: Record<Variant, string> = {
  base: "プロンプト版",
  tuned: "チューニング版",
};

const emptyMessages = (): Record<Variant, ChatMessage[]> => ({ base: [], tuned: [] });

export default function ChatRoom({ characterId }: { characterId: string }) {
  const [character, setCharacter] = useState<PublicCharacter | null>(null);
  const [loadError, setLoadError] = useState("");
  const [mode, setMode] = useState<Mode>("base");
  const [messages, setMessages] = useState(emptyMessages);
  const [sending, setSending] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    fetchCharacter(characterId)
      .then(setCharacter)
      .catch((e: Error) => setLoadError(e.message));
    return () => abortRef.current?.abort();
  }, [characterId]);

  const variants: Variant[] = mode === "compare" ? ["base", "tuned"] : [mode];

  // 指定したモデルの最後のメッセージ（返答中のもの）を書き換える
  const updateLast = (v: Variant, fn: (m: ChatMessage) => ChatMessage) =>
    setMessages((prev) => {
      const last = prev[v][prev[v].length - 1];
      // 「新しい会話」で消えた後に届いた返答は捨てる
      if (!last?.pending) return prev;
      return { ...prev, [v]: [...prev[v].slice(0, -1), fn(last)] };
    });

  const send = async (text: string) => {
    const sessionId = getSessionId();
    const abort = new AbortController();
    abortRef.current = abort;
    setSending(true);
    setMessages((prev) => {
      const next = { ...prev };
      for (const v of variants) {
        next[v] = [...prev[v], { role: "user", content: text }, { role: "model", content: "", pending: true }];
      }
      return next;
    });

    // 比較モードでは2つのモデルに同時に送る
    await Promise.all(
      variants.map(async (v) => {
        try {
          const { overlap, fallback } = await sendMessage(
            { characterId, sessionId, variant: v, message: text },
            (delta) => updateLast(v, (m) => ({ ...m, content: m.content + delta })),
            abort.signal,
          );
          updateLast(v, (m) => ({ ...m, pending: false, overlap, fallback }));
        } catch (e) {
          if (abort.signal.aborted) return;
          updateLast(v, (m) => ({ ...m, pending: false, error: (e as Error).message }));
        }
      }),
    );
    setSending(false);
  };

  const newConversation = () => {
    abortRef.current?.abort();
    resetSessionId();
    setMessages(emptyMessages());
    setSending(false);
  };

  if (loadError) {
    return <main className="p-4 text-red-600">キャラを読み込めませんでした（{loadError}）</main>;
  }
  if (!character) {
    return <main className="p-4 text-muted">読み込み中…</main>;
  }

  return (
    <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col px-4 py-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-xl font-bold">{character.name}</h1>
        <div className="flex flex-wrap items-center gap-2">
          <ModeSwitch mode={mode} onChange={setMode} hasTuned={character.hasTuned} disabled={sending} />
          <button
            type="button"
            onClick={newConversation}
            className="rounded border border-line px-3 py-1 text-sm hover:bg-surface"
          >
            新しい会話
          </button>
        </div>
      </div>
      {!character.hasTuned && (
        <p className="mt-2 text-xs text-muted">
          このキャラのチューニング版はまだありません。プロンプト版だけで会話できます。
        </p>
      )}

      <div className={`mt-4 grid min-h-0 flex-1 gap-4 ${variants.length === 2 ? "md:grid-cols-2" : ""}`}>
        {variants.map((v) => (
          <section key={v} className="flex min-h-[50vh] flex-col rounded-lg border border-line">
            {mode === "compare" && (
              <h2 className="border-b border-line px-3 py-2 text-sm font-bold">{LABELS[v]}</h2>
            )}
            <MessageList messages={messages[v]} characterName={character.name} />
          </section>
        ))}
      </div>

      <ChatInput onSend={send} disabled={sending} />
      <p className="mt-2 text-xs text-muted">
        会話の内容は記録され、サービスの改善のために管理者が閲覧することがあります。
      </p>
    </main>
  );
}

function ModeSwitch({
  mode,
  onChange,
  hasTuned,
  disabled,
}: {
  mode: Mode;
  onChange: (m: Mode) => void;
  hasTuned: boolean;
  disabled: boolean;
}) {
  const options: { value: Mode; label: string }[] = [
    { value: "base", label: LABELS.base },
    { value: "tuned", label: LABELS.tuned },
    { value: "compare", label: "比較" },
  ];
  return (
    <div role="radiogroup" aria-label="モデル" className="flex overflow-hidden rounded border border-line text-sm">
      {options.map((o) => {
        const needsTuned = o.value !== "base";
        const selected = mode === o.value;
        return (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={selected}
            disabled={disabled || (needsTuned && !hasTuned)}
            onClick={() => onChange(o.value)}
            className={`px-3 py-1 disabled:opacity-40 ${selected ? "bg-accent text-accent-foreground" : "hover:bg-surface"}`}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}
