// 管理系APIを Go に中継する。
// Cookie の管理トークンをここでヘッダーに付けるので、ブラウザの JS からトークンが見えない。
import type { NextRequest } from "next/server";
import { ADMIN_COOKIE } from "@/lib/admin-cookie";

type Ctx = { params: Promise<{ path: string[] }> };

const backendUrl = () => process.env.BACKEND_URL ?? "http://localhost:8080";

async function forward(request: NextRequest, { params }: Ctx) {
  const token = request.cookies.get(ADMIN_COOKIE)?.value;
  if (!token) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }

  const { path } = await params;
  const url = `${backendUrl()}/api/admin/${path.map(encodeURIComponent).join("/")}${request.nextUrl.search}`;
  const hasBody = request.method !== "GET" && request.method !== "DELETE";

  let res: Response;
  try {
    res = await fetch(url, {
      method: request.method,
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": request.headers.get("content-type") ?? "application/json",
      },
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
    });
  } catch {
    return Response.json({ error: "バックエンドに接続できません" }, { status: 502 });
  }

  // Go の返答をそのまま返す
  return new Response(res.body, {
    status: res.status,
    headers: { "Content-Type": res.headers.get("content-type") ?? "application/json" },
  });
}

export const GET = forward;
export const POST = forward;
export const PUT = forward;
export const DELETE = forward;
