// 利用規約とクレジット表記（つくよみちゃん会話AI育成計画の利用規約①に基づく）
export default function TermsPage() {
  return (
    <main className="mx-auto w-full max-w-3xl space-y-8 px-4 py-10 leading-relaxed">
      <h1 className="text-2xl font-bold">利用規約</h1>

      <section className="space-y-2">
        <h2 className="text-lg font-bold">禁止事項</h2>
        <p>本サービスの会話AIを、次の目的で使用することを禁止します。</p>
        <ul className="list-disc space-y-1 pl-6">
          <li>
            人を批判・攻撃すること（「批判・攻撃」の定義は、
            <a
              href="https://tyc.rei-yumesaki.net"
              className="underline"
              target="_blank"
              rel="noreferrer"
            >
              つくよみちゃんキャラクターライセンス
            </a>
            に準じます）
          </li>
          <li>特定の政治的立場・宗教・思想への賛同または反対を呼びかけること</li>
          <li>刺激の強い表現をゾーニングなしで公開すること</li>
        </ul>
      </section>

      <section className="space-y-2">
        <h2 className="text-lg font-bold">スクリーンショット・動画の投稿</h2>
        <p>
          会話画面のスクリーンショットやキャプチャ動画の投稿、会話AIが生成した会話を元にした作品の公開は、
          使用した会話AI（本サービス）とキャラクター（つくよみちゃん）が特定できる形で行ってください。
        </p>
      </section>

      <section className="space-y-2">
        <h2 className="text-lg font-bold">会話の記録</h2>
        <p>会話の内容は記録され、サービスの改善のために管理者が閲覧することがあります。</p>
      </section>

      <section className="space-y-2">
        <h2 className="text-lg font-bold">クレジット</h2>
        <p>
          会話AIの作成には、フリー素材キャラクター「つくよみちゃん」が無料公開している会話テキストデータセットを使用しています。
        </p>
        <p>
          ■つくよみちゃん会話AI育成計画
          <br />
          <a
            href="https://tyc.rei-yumesaki.net/material/kaiwa-ai/"
            className="underline"
            target="_blank"
            rel="noreferrer"
          >
            https://tyc.rei-yumesaki.net/material/kaiwa-ai/
          </a>
          <br />© Rei Yumesaki
        </p>
        <p>
          ■つくよみちゃん公式サイト
          <br />
          <a href="https://tyc.rei-yumesaki.net" className="underline" target="_blank" rel="noreferrer">
            https://tyc.rei-yumesaki.net
          </a>
        </p>
      </section>
    </main>
  );
}
