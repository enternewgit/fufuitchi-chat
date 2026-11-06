"use client";

import { useEffect, useState } from "react";
import { createCharacter, deleteCharacter, listCharacters, updateCharacter } from "@/lib/admin";
import type { Character } from "@/types";

const EMPTY: Character = {
  id: "",
  name: "",
  description: "",
  systemPrompt: "",
  baseModel: "gemini-3.5-flash",
  tunedModel: "",
};

export default function CharacterEditor() {
  const [chars, setChars] = useState<Character[]>([]);
  const [form, setForm] = useState<Character>(EMPTY);
  const [isNew, setIsNew] = useState(true);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const reload = async (selectId?: string) => {
    const cs = await listCharacters();
    setChars(cs);
    const selected = cs.find((c) => c.id === selectId);
    if (selected) {
      setForm(selected);
      setIsNew(false);
    }
  };

  useEffect(() => {
    listCharacters()
      .then((cs) => {
        setChars(cs);
        if (cs.length > 0) {
          setForm(cs[0]);
          setIsNew(false);
        }
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  const select = (c: Character | null) => {
    setForm(c ?? EMPTY);
    setIsNew(c === null);
    setMessage("");
    setError("");
  };

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage("");
    setError("");
    try {
      await (isNew ? createCharacter(form) : updateCharacter(form));
      await reload(form.id);
      setMessage("保存しました");
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const remove = async () => {
    if (!confirm(`「${form.name}」を削除しますか？`)) return;
    try {
      await deleteCharacter(form.id);
      await reload();
      select(null);
      setMessage("削除しました");
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const field = (key: keyof Character) => ({
    value: form[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm({ ...form, [key]: e.target.value }),
  });

  return (
    <div className="mt-4 grid gap-6 md:grid-cols-[200px_1fr]">
      <aside>
        <ul className="space-y-1 text-sm">
          {chars.map((c) => (
            <li key={c.id}>
              <button
                type="button"
                onClick={() => select(c)}
                className={`w-full rounded px-2 py-1 text-left hover:bg-surface ${
                  !isNew && form.id === c.id ? "bg-surface font-bold" : ""
                }`}
              >
                {c.name}
              </button>
            </li>
          ))}
        </ul>
        <button
          type="button"
          onClick={() => select(null)}
          className="mt-3 w-full rounded border border-line px-2 py-1 text-sm hover:bg-surface"
        >
          ＋ 新しいキャラ
        </button>
      </aside>

      <form onSubmit={save} className="space-y-3">
        <p className="text-xs text-muted">
          変更はサーバーのメモリにだけ保存され、再起動すると data/characters/ の内容に戻ります。
        </p>
        <Field label="ID（英小文字・数字・-・_）">
          <input {...field("id")} readOnly={!isNew} className={input} required />
        </Field>
        <Field label="名前">
          <input {...field("name")} className={input} required />
        </Field>
        <Field label="紹介文（キャラ選択画面に表示）">
          <input {...field("description")} className={input} />
        </Field>
        <Field label="システムプロンプト（ユーザーには見えない）">
          <textarea {...field("systemPrompt")} rows={12} className={`${input} font-mono text-sm`} />
        </Field>
        <Field label="ベースモデル（プロンプト版）">
          <input {...field("baseModel")} className={input} required />
        </Field>
        <Field label="チューニング済みモデル（空なら未作成）">
          <input {...field("tunedModel")} className={input} placeholder="projects/…/endpoints/…" />
        </Field>
        {message && <p className="text-sm text-green-600">{message}</p>}
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex gap-2">
          <button type="submit" className="rounded bg-accent px-4 py-2 text-accent-foreground">
            {isNew ? "追加" : "保存"}
          </button>
          {!isNew && (
            <button type="button" onClick={remove} className="rounded border border-line px-4 py-2 text-red-600">
              削除
            </button>
          )}
        </div>
      </form>
    </div>
  );
}

const input = "mt-1 w-full rounded border border-line bg-background px-3 py-2 read-only:opacity-60";

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block text-sm">
      {label}
      {children}
    </label>
  );
}
