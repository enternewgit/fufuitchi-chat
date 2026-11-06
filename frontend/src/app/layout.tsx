import type { Metadata } from "next";
import Link from "next/link";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "解釈不不一致チャット",
  description: "そのキャラ、そんなこと言わない。を、なくす。ファインチューニングでキャラの「らしさ」を再現するチャット",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="ja"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <header className="border-b border-line">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
            <Link href="/" className="font-bold">
              解釈不不一致チャット
            </Link>
          </div>
        </header>
        <div className="flex flex-1 flex-col">{children}</div>
        <footer className="border-t border-line text-xs text-muted">
          <div className="mx-auto max-w-5xl space-y-1 px-4 py-4">
            <p>
              会話AIの作成には、フリー素材キャラクター「つくよみちゃん」が無料公開している会話テキストデータセットを使用しています。
            </p>
            <p>
              ■
              <a
                href="https://tyc.rei-yumesaki.net/material/kaiwa-ai/"
                className="underline"
                target="_blank"
                rel="noreferrer"
              >
                つくよみちゃん会話AI育成計画
              </a>{" "}
              © Rei Yumesaki
            </p>
            <p>
              ■
              <a
                href="https://tyc.rei-yumesaki.net"
                className="underline"
                target="_blank"
                rel="noreferrer"
              >
                つくよみちゃん公式サイト
              </a>
            </p>
            <p>
              <Link href="/terms" className="underline">
                利用規約
              </Link>
            </p>
          </div>
        </footer>
      </body>
    </html>
  );
}
