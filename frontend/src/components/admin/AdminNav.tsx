"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { getMe } from "@/lib/admin";

const LINKS = [
  { href: "/admin/characters", label: "キャラ設定" },
  { href: "/admin/logs", label: "会話ログ" },
];

export default function AdminNav() {
  const pathname = usePathname();
  const router = useRouter();
  const [email, setEmail] = useState("");

  useEffect(() => {
    getMe()
      .then((me) => setEmail(me.email))
      .catch(() => {});
  }, []);

  const logout = async () => {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/admin/login");
  };

  return (
    <nav className="flex flex-wrap items-center gap-4 border-b border-line pb-3 text-sm">
      <span className="font-bold">管理画面</span>
      {LINKS.map((l) => (
        <Link
          key={l.href}
          href={l.href}
          className={pathname === l.href ? "font-bold underline" : "hover:underline"}
        >
          {l.label}
        </Link>
      ))}
      <span className="ml-auto text-muted">{email}</span>
      <button type="button" onClick={logout} className="text-muted hover:underline">
        ログアウト
      </button>
    </nav>
  );
}
