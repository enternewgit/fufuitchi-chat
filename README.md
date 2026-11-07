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
cp .env.example .env          # GEMINI_API_KEY、GOOGLE_CLIENT_ID、ADMIN_EMAILS を埋める
docker compose up --build
```

- フロントエンド: http://localhost:3000
- 管理画面: http://localhost:3000/admin （`ADMIN_EMAILS` に入れた Google アカウントでログイン）
- バックエンド: http://localhost:8080/healthz

### 管理者ログイン（Google）

ブラウザの「Sign in with Google」が受け取った ID トークンを Go が検証し、`ADMIN_EMAILS` に入っているアドレスだけ管理者として通します。
クライアントシークレットは使わないので、サーバーに秘密を置く必要はありません。セッションはメモリに保存され、backend を再起動すると全員ログアウトになります。

1. Google Cloud Console の「API とサービス」→「認証情報」で、OAuth クライアントID（種類: ウェブアプリケーション）を作る
2. 「承認済みの JavaScript 生成元」に `http://localhost` と `http://localhost:3000` を追加する（公開するならそのドメインも）
3. `.env` の `GOOGLE_CLIENT_ID` にクライアントID、`ADMIN_EMAILS` に自分のメールアドレスを入れる
4. `docker compose up --build`（クライアントIDは frontend のビルド時に埋め込まれる）

### 保存先

`.env` の `STORAGE` で選べます。

| `STORAGE` | 会話ログ・管理画面での編集・ログイン状態 |
|---|---|
| `memory`（初期値） | backend のメモリ。再起動すると消える |
| `firestore` | Firestore に保存。再起動しても残る |

Firestore を使うときは、`gcloud auth application-default login` を済ませ、`GOOGLE_CLOUD_PROJECT` を設定して、`docker-compose.vertex.yml` を重ねて起動します。
キャラ設定は起動時に `data/characters/` から読み、Firestore にまだないキャラだけを追加します（管理画面での編集は上書きしません）。

Firestore のコレクション:

| コレクション | 中身 |
|---|---|
| `characters/{キャラID}` | キャラ設定 |
| `messages/{自動ID}` | 会話の1発言 |
| `sessions/{セッションID}` | 会話セッションの集計（管理画面の一覧用） |
| `adminSessions/{トークンのハッシュ}` | 管理者のログインセッション（トークンそのものは保存しない） |

## 学習データ

[つくよみちゃん会話AI育成計画](https://tyc.rei-yumesaki.net/material/kaiwa-ai/) の会話テキストデータセットを使っています。
データの再配布は禁止されているので、各自で配布ページから Excel をダウンロードしてください。

```bash
# 1. Excel を data/raw/kaiwa-ai.xlsx に置く
# 2. Vertex AI のチューニング用 JSONL に整形する（data/processed/train.jsonl, validation.jsonl）
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -v "$PWD":/work -w /work/backend \
  golang:1.26-alpine go run ./cmd/prepare-dataset
```

整形後のデータは、チャット中に「学習データとの最長一致」（お返事と最長で何文字連続して同じか）を計算するのにも使います（backend の再起動で読み込み直します）。

## チューニング済みモデルを使う

チューニング済みモデルは Vertex AI でしか呼べません。チューニングジョブは `scripts/tuning.sh` で作れます（Gemini 3 は Console からはチューニングできない）。

1. `gcloud auth application-default login`
2. `.env` の `GEMINI_API_KEY` を空にして、`GOOGLE_CLOUD_PROJECT` を埋め、`GOOGLE_CLOUD_LOCATION=us` にする
   （Gemini 3 のチューニング済みモデルは `us` マルチリージョンにある。`global` からは呼べない）
3. `.env` の `TUNED_MODEL_<キャラID>` にチューニング済みモデルのエンドポイントを入れる（`scripts/tuning.sh status <ジョブID>` の `tunedModel.endpoint`）。
   キャラシートの `tunedModel` より優先されるので、プロジェクト固有の値をリポジトリに入れずに済む
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
