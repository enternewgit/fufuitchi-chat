// 管理系APIを Go に中継する。
// Cookie の管理トークンをここでヘッダーに付けるので、ブラウザの JS からトークンが見えない。
import type { NextRequest } from "next/server";

type Ctx = { params: Promise<{ path: string[] }> };

async function forward(request: NextRequest, { params }: Ctx) {
  const { path } = await params;
  // TODO:
  //   1. Cookie から管理トークンを取り出す
  //   2. `${process.env.BACKEND_URL}/api/admin/${path.join("/")}` に
  //      Authorization: Bearer <token> を付けて転送
  //   3. Go の返答をそのまま返す
  return Response.json({ error: "not implemented", path }, { status: 501 });
}

export const GET = forward;
export const POST = forward;
export const PUT = forward;
export const DELETE = forward;
