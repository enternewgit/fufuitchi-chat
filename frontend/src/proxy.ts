// Next.js 16 では middleware.ts が proxy.ts に改名された
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function proxy(request: NextRequest) {
  // TODO: 管理トークンの Cookie がなければ /admin/login に飛ばす
  //   ※ 画面を隠すだけ。本当のチェックは Go 側でやる
  return NextResponse.next();
}

export const config = {
  matcher: ["/admin/((?!login).*)"],
};
