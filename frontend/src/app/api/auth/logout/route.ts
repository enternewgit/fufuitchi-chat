import { NextResponse, type NextRequest } from "next/server";
import { ADMIN_COOKIE } from "@/lib/admin-cookie";

const backendUrl = () => process.env.BACKEND_URL ?? "http://localhost:8080";

export async function POST(request: NextRequest) {
  const token = request.cookies.get(ADMIN_COOKIE)?.value;
  if (token) {
    // Go 側のセッションも消す（失敗しても Cookie は消す）
    await fetch(`${backendUrl()}/api/auth/logout`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    }).catch(() => {});
  }
  const out = NextResponse.json({ ok: true });
  out.cookies.delete(ADMIN_COOKIE);
  return out;
}
