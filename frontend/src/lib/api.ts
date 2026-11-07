// Go API を呼ぶ関数をここにまとめる（ブラウザから直接呼ぶ）
import type { ChatEvent, Overlap, PublicCharacter, Variant } from "@/types";

export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function errorMessage(res: Response): Promise<string> {
  try {
    const body = await res.json();
    if (typeof body?.error === "string") return body.error;
  } catch {
    // JSON でなければステータスだけ返す
  }
  return `HTTP ${res.status}`;
}

export async function fetchCharacters(): Promise<PublicCharacter[]> {
  const res = await fetch(`${API_URL}/api/characters`);
  if (!res.ok) throw new Error(await errorMessage(res));
  return res.json();
}

export async function fetchCharacter(id: string): Promise<PublicCharacter> {
  const res = await fetch(`${API_URL}/api/characters/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error(await errorMessage(res));
  return res.json();
}

export type SendMessageInput = {
  characterId: string;
  sessionId: string;
  variant: Variant;
  message: string;
};

// sendMessage はメッセージを送り、SSE で届く返答を onDelta に少しずつ渡す。
// 返答が終わると学習データとの重なり（学習データがない場合は undefined）を返す。
export async function sendMessage(
  input: SendMessageInput,
  onDelta: (text: string) => void,
  signal?: AbortSignal,
): Promise<{ overlap?: Overlap }> {
  const res = await fetch(`${API_URL}/api/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
    signal,
  });
  if (!res.ok || !res.body) throw new Error(await errorMessage(res));

  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
  let buf = "";
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buf += value;
    // イベントは空行（\n\n）で区切られている
    let sep: number;
    while ((sep = buf.indexOf("\n\n")) >= 0) {
      const block = buf.slice(0, sep);
      buf = buf.slice(sep + 2);
      for (const line of block.split("\n")) {
        if (!line.startsWith("data: ")) continue;
        const ev = JSON.parse(line.slice(6)) as ChatEvent;
        if (ev.type === "delta") onDelta(ev.text);
        else if (ev.type === "done") return { overlap: ev.overlap };
        else if (ev.type === "error") throw new Error(ev.error);
      }
    }
  }
  throw new Error("返答が途中で切れました");
}
