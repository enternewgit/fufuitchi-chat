// チャット画面（比較モード含む）
import { Suspense } from "react";
import ChatRoom from "@/components/chat/ChatRoom";

type Params = Promise<{ characterId: string }>;

export default function ChatPage({ params }: { params: Params }) {
  // cacheComponents が有効なので、params の読み取りは Suspense の内側で行う
  return (
    <Suspense fallback={<main className="p-4 text-muted">読み込み中…</main>}>
      <Chat params={params} />
    </Suspense>
  );
}

async function Chat({ params }: { params: Params }) {
  const { characterId } = await params;
  return <ChatRoom characterId={decodeURIComponent(characterId)} />;
}
