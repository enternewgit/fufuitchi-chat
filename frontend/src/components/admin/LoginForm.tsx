"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

export default function LoginForm() {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    });
    setBusy(false);
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      setError(body?.error ?? "ログインに失敗しました");
      return;
    }
    router.push("/admin/characters");
  };

  return (
    <form onSubmit={submit} className="mt-6 space-y-3">
      <label className="block text-sm">
        管理トークン（.env の ADMIN_TOKEN）
        <input
          type="password"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          autoComplete="current-password"
          className="mt-1 w-full rounded border border-line bg-background px-3 py-2"
        />
      </label>
      {error && <p className="text-sm text-red-600">{error}</p>}
      <button
        type="submit"
        disabled={busy || !token}
        className="w-full rounded bg-accent py-2 text-accent-foreground disabled:opacity-40"
      >
        ログイン
      </button>
    </form>
  );
}
