"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";

const LINKS = [
  { href: "/admin/characters", label: "キャラ設定" },
  { href: "/admin/logs", label: "会話ログ" },
];

export default function AdminNav() {
  const pathname = usePathname();
  const router = useRouter();

  const logout = async () => {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/admin/login");
  };

  return (
    <nav className="flex items-center gap-4 border-b border-line pb-3 text-sm">
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
      <button type="button" onClick={logout} className="ml-auto text-muted hover:underline">
        ログアウト
      </button>
    </nav>
  );
}
