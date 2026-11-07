"use client";

import { useEffect, useState } from "react";
import { getMe } from "@/lib/admin";

// 保存先がメモリのときだけ、再起動でデータが消えることを知らせる
export default function MemoryStorageNote({ children }: { children: React.ReactNode }) {
  const [isMemory, setIsMemory] = useState(false);

  useEffect(() => {
    getMe()
      .then((me) => setIsMemory(me.storage === "memory"))
      .catch(() => {});
  }, []);

  if (!isMemory) return null;
  return <p className="text-xs text-muted">{children}</p>;
}
