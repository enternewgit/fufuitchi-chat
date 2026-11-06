// 管理画面から呼ぶ関数。Next.js の /api/admin/* を経由して Go に中継する。
// トークンは httpOnly Cookie にあり、ここ（ブラウザの JS）からは見えない。
import type { Character, Message, SessionSummary } from "@/types";

async function adminFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/admin/${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (res.status === 401) {
    // React の外の関数なので router が使えない。Cookie の状態を確実に反映するため全画面で移動する
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.href = "/admin/login";
    throw new Error("ログインが必要です");
  }
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const getMe = () => adminFetch<{ email: string; expiresAt: string }>("me");

export const listCharacters = () => adminFetch<Character[]>("characters");

export const createCharacter = (c: Character) =>
  adminFetch<Character>("characters", { method: "POST", body: JSON.stringify(c) });

export const updateCharacter = (c: Character) =>
  adminFetch<Character>(`characters/${encodeURIComponent(c.id)}`, {
    method: "PUT",
    body: JSON.stringify(c),
  });

export const deleteCharacter = (id: string) =>
  adminFetch<void>(`characters/${encodeURIComponent(id)}`, { method: "DELETE" });

export type LogFilter = { characterId?: string; from?: string; to?: string };

export const listLogs = (f: LogFilter) => {
  const q = new URLSearchParams();
  if (f.characterId) q.set("characterId", f.characterId);
  if (f.from) q.set("from", f.from);
  if (f.to) q.set("to", f.to);
  return adminFetch<SessionSummary[]>(`logs?${q}`);
};

export const getLog = (sessionId: string) =>
  adminFetch<Message[]>(`logs/${encodeURIComponent(sessionId)}`);
