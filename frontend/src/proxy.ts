// Next.js 16 では middleware.ts が proxy.ts に改名された
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { ADMIN_COOKIE } from "@/lib/admin-cookie";

export function proxy(request: NextRequest) {
  // 管理トークンの Cookie がなければ /admin/login に飛ばす
  // ※ 画面を隠すだけ。本当のチェックは Go 側でやる
  if (!request.cookies.has(ADMIN_COOKIE)) {
    return NextResponse.redirect(new URL("/admin/login", request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/admin", "/admin/((?!login).*)"],
};
