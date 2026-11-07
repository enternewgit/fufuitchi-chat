"use client";

import { useEffect, useState } from "react";
import { getLog, listCharacters, listLogs, type LogFilter } from "@/lib/admin";
import type { Character, Message, SessionSummary } from "@/types";
import { formatOverlap } from "@/components/chat/MessageList";

const VARIANT_LABEL = { base: "プロンプト版", tuned: "チューニング版" } as const;

const formatTime = (s: string) =>
  new Date(s).toLocaleString("ja-JP", { dateStyle: "short", timeStyle: "short" });

export default function LogViewer() {
  const [chars, setChars] = useState<Character[]>([]);
  const [filter, setFilter] = useState<LogFilter>({});
  const [sessions, setSessions] = useState<SessionSummary[] | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([listCharacters(), listLogs({})])
      .then(([cs, ss]) => {
        setChars(cs);
        setSessions(ss);
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  const search = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      setSessions(await listLogs(filter));
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const open = async (sessionId: string) => {
    setSelected(sessionId);
    setError("");
    try {
      setMessages(await getLog(sessionId));
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const charName = (id: string) => chars.find((c) => c.id === id)?.name ?? id;

  return (
    <div className="mt-4 space-y-4">
      <form onSubmit={search} className="flex flex-wrap items-end gap-3 text-sm">
        <label>
          キャラ
          <select
            value={filter.characterId ?? ""}
            onChange={(e) => setFilter({ ...filter, characterId: e.target.value })}
            className={input}
          >
            <option value="">すべて</option>
            {chars.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          開始日
          <input
            type="date"
            value={filter.from ?? ""}
            onChange={(e) => setFilter({ ...filter, from: e.target.value })}
            className={input}
          />
        </label>
        <label>
          終了日
          <input
            type="date"
            value={filter.to ?? ""}
            onChange={(e) => setFilter({ ...filter, to: e.target.value })}
            className={input}
          />
        </label>
        <button type="submit" className="rounded bg-accent px-4 py-2 text-accent-foreground">
          絞り込み
        </button>
      </form>
      {error && <p className="text-sm text-red-600">{error}</p>}
      <p className="text-xs text-muted">会話ログはサーバーのメモリにだけ保存され、再起動すると消えます。</p>

      <div className="grid gap-6 md:grid-cols-[320px_1fr]">
        <ul className="space-y-1 text-sm">
          {sessions?.length === 0 && <li className="text-muted">会話ログはありません</li>}
          {sessions?.map((s) => (
            <li key={s.sessionId}>
              <button
                type="button"
                onClick={() => open(s.sessionId)}
                className={`w-full rounded px-2 py-1 text-left hover:bg-surface ${
                  selected === s.sessionId ? "bg-surface" : ""
                }`}
              >
                <div className="font-bold">{charName(s.characterId)}</div>
                <div className="text-xs text-muted">
                  {formatTime(s.firstAt)} 〜 {formatTime(s.lastAt)}・{s.messageCount}件
                </div>
              </button>
            </li>
          ))}
        </ul>

        <div className="space-y-2 text-sm">
          {selected === null && <p className="text-muted">左の一覧から会話を選んでください</p>}
          {selected !== null &&
            messages.map((m, i) => (
              <div key={i} className="rounded border border-line p-2">
                <div className="text-xs text-muted">
                  {formatTime(m.createdAt)}・{m.role === "user" ? "ユーザー" : charName(m.characterId)}
                  （{VARIANT_LABEL[m.variant]}）
                  {m.overlap && `・${formatOverlap(m.overlap)}`}
                </div>
                <p className="mt-1 whitespace-pre-wrap">{m.content}</p>
              </div>
            ))}
        </div>
      </div>
    </div>
  );
}

const input = "mt-1 block rounded border border-line bg-background px-2 py-1.5";
