# fufuitchi-chat

**解釈不不一致チャット** — そのキャラ、そんなこと言わない。を、なくす。

プロンプトだけでは制御しきれないキャラの「らしさ」を、ファインチューニングで再現するキャラクターチャットbot。

## 構成

| ディレクトリ | 内容 |
|---|---|
| `frontend/` | Next.js（チャットUI、比較モード、管理画面） |
| `backend/` | Go（API、会話履歴、モデル呼び出し、学習データとの一致率） |
| `data/characters/` | キャラシート（1ファイル1キャラの JSON） |
| `data/raw/`, `data/processed/` | 学習データ（再配布禁止のためコミットしない） |

## セットアップ

```bash
cp .env.example .env          # GEMINI_API_KEY と ADMIN_TOKEN を埋める
docker compose up --build
```

- フロントエンド: http://localhost:3000
- 管理画面: http://localhost:3000/admin （`.env` の `ADMIN_TOKEN` でログイン）
- バックエンド: http://localhost:8080/healthz

キャラ設定と会話ログはメモリに保存されるので、backend を再起動すると `data/characters/` の内容に戻り、ログは消えます。

## 学習データ

[つくよみちゃん会話AI育成計画](https://tyc.rei-yumesaki.net/material/kaiwa-ai/) の会話テキストデータセットを使っています。
データの再配布は禁止されているので、各自で配布ページから Excel をダウンロードしてください。

```bash
# 1. Excel を data/raw/kaiwa-ai.xlsx に置く
# 2. Vertex AI のチューニング用 JSONL に整形する（data/processed/train.jsonl, validation.jsonl）
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -v "$PWD":/work -w /work/backend \
  golang:1.26-alpine go run ./cmd/prepare-dataset
```

整形後のデータは、チャット中に「学習データとの一致率」を計算するのにも使います（backend の再起動で読み込み直します）。

## チューニング済みモデルを使う

チューニング済みモデルは Vertex AI でしか呼べません。

1. `gcloud auth application-default login`
2. `.env` の `GEMINI_API_KEY` を空にして、`GOOGLE_CLOUD_PROJECT` を埋める
3. キャラシートの `tunedModel` にモデルIDを入れる（管理画面からも設定可）
4. 起動する
   ```bash
   docker compose -f docker-compose.yml -f docker-compose.vertex.yml up --build
   ```

## テスト

```bash
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -v "$PWD":/work -w /work/backend \
  golang:1.26-alpine go test ./...
```

## クレジット

会話AIの作成には、フリー素材キャラクター「つくよみちゃん」が無料公開している会話テキストデータセットを使用しています。

■つくよみちゃん会話AI育成計画
https://tyc.rei-yumesaki.net/material/kaiwa-ai/
© Rei Yumesaki
