# fufuitchi-chat

**解釈不不一致チャット** — そのキャラ、そんなこと言わない。を、なくす。

プロンプトだけでは制御しきれないキャラの「らしさ」を、ファインチューニングで再現するキャラクターチャットbot。

## 構成

| ディレクトリ | 内容 |
|---|---|
| `frontend/` | Next.js（チャットUI、比較モード、管理画面） |
| `backend/` | Go（API、会話履歴、モデル呼び出し、丸写し検出） |
| `data/` | キャラシートと学習データ（学習データはコミットしない） |

## セットアップ

```bash
cp .env.example .env          # 値を埋める
gcloud auth application-default login
docker compose up --build
```

- フロントエンド: http://localhost:3000
- バックエンド: http://localhost:8080/healthz

## データ整形

```bash
cd backend
go run ./cmd/prepare-dataset
```
