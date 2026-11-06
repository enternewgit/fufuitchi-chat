import { NextResponse } from "next/server";
import { ADMIN_COOKIE } from "@/lib/admin-cookie";

export async function POST() {
  const out = NextResponse.json({ ok: true });
  out.cookies.delete(ADMIN_COOKIE);
  return out;
}
