"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { fetchCharacters } from "@/lib/api";
import type { PublicCharacter } from "@/types";

export default function CharacterList() {
  const [chars, setChars] = useState<PublicCharacter[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    fetchCharacters()
      .then(setChars)
      .catch((e: Error) => setError(e.message));
  }, []);

  if (error) return <p className="text-red-600">キャラ一覧を取得できませんでした（{error}）</p>;
  if (!chars) return <p className="text-muted">読み込み中…</p>;
  if (chars.length === 0) return <p className="text-muted">キャラがまだ登録されていません</p>;

  return (
    <ul className="grid gap-4 sm:grid-cols-2">
      {chars.map((c) => (
        <li key={c.id}>
          <Link
            href={`/chat/${encodeURIComponent(c.id)}`}
            className="block rounded-lg border border-line p-4 hover:bg-surface"
          >
            <div className="flex items-center gap-2">
              <span className="font-bold">{c.name}</span>
              {c.hasTuned && (
                <span className="rounded bg-accent px-1.5 py-0.5 text-xs text-accent-foreground">
                  チューニング版あり
                </span>
              )}
            </div>
            <p className="mt-1 text-sm text-muted">{c.description}</p>
          </Link>
        </li>
      ))}
    </ul>
  );
}
