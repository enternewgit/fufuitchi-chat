// チャット画面（比較モード含む）
import { Suspense } from "react";

type Params = Promise<{ characterId: string }>;

export default function ChatPage({ params }: { params: Params }) {
  // cacheComponents が有効なので、params の読み取りは Suspense の内側で行う
  return (
    <Suspense fallback={<main>loading...</main>}>
      <Chat params={params} />
    </Suspense>
  );
}

async function Chat({ params }: { params: Params }) {
  const { characterId } = await params;

  // TODO:
  //   - メッセージ一覧と入力欄（components/chat）
  //   - ストリーミングで届く返答の逐次表示
  //   - プロンプト版 / チューニング版の比較モード
  //   - 「会話は記録されます」の表示
  return (
    <main>
      <p>character: {characterId}</p>
    </main>
  );
}
