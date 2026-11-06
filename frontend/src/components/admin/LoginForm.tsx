"use client";

import Script from "next/script";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";

// Google Identity Services（https://accounts.google.com/gsi/client）のうち使う部分だけの型
type GoogleIdentity = {
  accounts: {
    id: {
      initialize(config: {
        client_id: string;
        callback: (res: { credential: string }) => void;
      }): void;
      renderButton(el: HTMLElement, options: Record<string, string>): void;
    };
  };
};

declare global {
  interface Window {
    google?: GoogleIdentity;
  }
}

const CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? "";

export default function LoginForm() {
  const router = useRouter();
  const buttonRef = useRef<HTMLDivElement>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  // Google から ID トークンを受け取ったら、Next.js 経由で Go に確認してもらう
  const onCredential = async ({ credential }: { credential: string }) => {
    setBusy(true);
    setError("");
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ credential }),
    });
    setBusy(false);
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      setError(body?.error ?? "ログインに失敗しました");
      return;
    }
    router.push("/admin/characters");
  };

  const renderButton = () => {
    if (!window.google || !buttonRef.current) return;
    window.google.accounts.id.initialize({ client_id: CLIENT_ID, callback: onCredential });
    window.google.accounts.id.renderButton(buttonRef.current, {
      theme: "outline",
      size: "large",
      text: "signin_with",
      locale: "ja",
    });
  };

  if (!CLIENT_ID) {
    return (
      <p className="mt-6 text-sm text-red-600">
        NEXT_PUBLIC_GOOGLE_CLIENT_ID が設定されていません。.env に設定して frontend をビルドし直してください。
      </p>
    );
  }

  return (
    <div className="mt-6 space-y-3">
      {/* onReady はスクリプトの読み込み済みでも、表示されるたびに呼ばれる */}
      <Script src="https://accounts.google.com/gsi/client" strategy="afterInteractive" onReady={renderButton} />
      <p className="text-sm text-muted">許可された Google アカウントでログインしてください。</p>
      <div ref={buttonRef} />
      {busy && <p className="text-sm text-muted">確認中…</p>}
      {error && <p className="text-sm text-red-600">{error}</p>}
    </div>
  );
}
