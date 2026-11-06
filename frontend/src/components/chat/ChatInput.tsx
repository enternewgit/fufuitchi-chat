"use client";

import { useState } from "react";

const MAX_LENGTH = 1000; // Go 側の上限と合わせる

export default function ChatInput({
  onSend,
  disabled,
}: {
  onSend: (text: string) => void;
  disabled: boolean;
}) {
  const [text, setText] = useState("");

  const submit = () => {
    const t = text.trim();
    if (!t || disabled) return;
    onSend(t);
    setText("");
  };

  return (
    <form
      className="mt-4 flex gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
    >
      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          // Enter で送信、Shift+Enter で改行（日本語の変換確定中は送信しない）
          if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
            e.preventDefault();
            submit();
          }
        }}
        maxLength={MAX_LENGTH}
        rows={2}
        placeholder="メッセージを入力（Shift+Enter で改行）"
        aria-label="メッセージ"
        className="flex-1 resize-none rounded-lg border border-line bg-background px-3 py-2"
      />
      <button
        type="submit"
        disabled={disabled || !text.trim()}
        className="rounded-lg bg-accent px-4 text-accent-foreground disabled:opacity-40"
      >
        送信
      </button>
    </form>
  );
}
