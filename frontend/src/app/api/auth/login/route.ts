// 管理者ログイン。トークンが正しいか Go に確認してから、httpOnly Cookie に保存する。
import { NextResponse, type NextRequest } from "next/server";
import { ADMIN_COOKIE } from "@/lib/admin-cookie";

const backendUrl = () => process.env.BACKEND_URL ?? "http://localhost:8080";

export async function POST(request: NextRequest) {
  const body = await request.json().catch(() => null);
  const token = typeof body?.token === "string" ? body.token.trim() : "";
  if (!token) {
    return NextResponse.json({ error: "トークンを入力してください" }, { status: 400 });
  }

  let res: Response;
  try {
    res = await fetch(`${backendUrl()}/api/admin/characters`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
  } catch {
    return NextResponse.json({ error: "バックエンドに接続できません" }, { status: 502 });
  }
  if (res.status === 401) {
    return NextResponse.json({ error: "トークンが違います" }, { status: 401 });
  }
  if (!res.ok) {
    return NextResponse.json({ error: `確認に失敗しました（HTTP ${res.status}）` }, { status: 502 });
  }

  const out = NextResponse.json({ ok: true });
  out.cookies.set(ADMIN_COOKIE, token, {
    httpOnly: true,
    sameSite: "strict",
    secure: request.nextUrl.protocol === "https:",
    path: "/",
    maxAge: 60 * 60 * 24 * 7,
  });
  return out;
}
