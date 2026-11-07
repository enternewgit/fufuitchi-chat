// セッションIDの生成・保持（ブラウザの localStorage）
const KEY = "fufuitchi-session-id";

export function getSessionId(): string {
  try {
    let id = localStorage.getItem(KEY);
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem(KEY, id);
    }
    return id;
  } catch {
    // localStorage が使えない環境ではその場限りのIDにする
    return crypto.randomUUID();
  }
}

// resetSessionId は新しい会話を始めるときに呼ぶ。
export function resetSessionId(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    // 使えない環境では何もしない
  }
}

// ユーザーID（ブラウザごと）。「新しい会話」を始めても変わらないので、
// つくよみちゃんが前の会話を覚えているのに使う
const USER_KEY = "fufuitchi-user-id";

export function getUserId(): string {
  try {
    let id = localStorage.getItem(USER_KEY);
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem(USER_KEY, id);
    }
    return id;
  } catch {
    // localStorage が使えない環境では前の会話を引き継がない
    return "";
  }
}
