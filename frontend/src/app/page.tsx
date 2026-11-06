// キャラ選択画面
import CharacterList from "@/components/CharacterList";

export default function Home() {
  return (
    <main className="mx-auto w-full max-w-5xl px-4 py-10">
      <h1 className="text-2xl font-bold">解釈不不一致チャット</h1>
      <p className="mt-2 text-muted">そのキャラ、そんなこと言わない。を、なくす。</p>
      <h2 className="mt-8 mb-3 font-bold">話したいキャラを選んでください</h2>
      <CharacterList />
    </main>
  );
}
