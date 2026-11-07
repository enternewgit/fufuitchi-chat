// 管理者ログイン。Google の ID トークンを Go に渡して確認してもらい、
// 発行されたセッショントークンを httpOnly Cookie に保存する。
import { NextResponse, type NextRequest } from "next/server";
import { ADMIN_COOKIE } from "@/lib/admin-cookie";

const backendUrl = () => process.env.BACKEND_URL ?? "http://localhost:8080";

export async function POST(request: NextRequest) {
  const body = await request.json().catch(() => null);
  const credential = typeof body?.credential === "string" ? body.credential : "";
  if (!credential) {
    return NextResponse.json({ error: "credential が空です" }, { status: 400 });
  }

  let res: Response;
  try {
    res = await fetch(`${backendUrl()}/api/auth/google`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ credential }),
      cache: "no-store",
    });
  } catch {
    return NextResponse.json({ error: "バックエンドに接続できません" }, { status: 502 });
  }
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    return NextResponse.json({ error: data?.error ?? `HTTP ${res.status}` }, { status: res.status });
  }

  const out = NextResponse.json({ email: data.email });
  out.cookies.set(ADMIN_COOKIE, data.token, {
    httpOnly: true,
    sameSite: "strict",
    // Cloud Run などプロキシの後ろでは https かどうか分からないので、COOKIE_SECURE で強制できるようにする
    secure: process.env.COOKIE_SECURE === "true" || request.nextUrl.protocol === "https:",
    path: "/",
    expires: new Date(data.expiresAt),
  });
  return out;
}
