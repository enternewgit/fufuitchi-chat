#!/usr/bin/env bash
# Cloud Run にデプロイする / 片付けるスクリプト。
#
#   scripts/deploy.sh setup     # API の有効化、イメージ置き場、サービスアカウントを用意する（初回だけ）
#   scripts/deploy.sh deploy    # イメージを作って backend と frontend をデプロイする
#   scripts/deploy.sh teardown  # デプロイしたものを全部消す（Firestore とチューニング済みモデルは残す）
#
# 値は .env から読む（STORAGE=firestore 前提）。学習データはイメージに入れない。
set -euo pipefail
cd "$(dirname "$0")/.."

env_get() { grep "^$1=" .env | head -1 | cut -d= -f2-; }

PROJECT="$(env_get GOOGLE_CLOUD_PROJECT)"
REGION="${REGION:-asia-northeast1}"
REPO=fufuitchi
AR="${REGION}-docker.pkg.dev/${PROJECT}/${REPO}"
BACKEND=fufuitchi-backend
FRONTEND=fufuitchi-frontend
BACKEND_SA="${BACKEND}@${PROJECT}.iam.gserviceaccount.com"
FRONTEND_SA="${FRONTEND}@${PROJECT}.iam.gserviceaccount.com"
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT" --format='value(projectNumber)')"
# Cloud Run の URL は「サービス名-プロジェクト番号.リージョン.run.app」で決まるので、先に組み立てられる
BACKEND_URL="https://${BACKEND}-${PROJECT_NUMBER}.${REGION}.run.app"
FRONTEND_URL="https://${FRONTEND}-${PROJECT_NUMBER}.${REGION}.run.app"

setup() {
  gcloud services enable run.googleapis.com artifactregistry.googleapis.com --project "$PROJECT"
  gcloud artifacts repositories describe "$REPO" --location "$REGION" --project "$PROJECT" >/dev/null 2>&1 ||
    gcloud artifacts repositories create "$REPO" --repository-format docker --location "$REGION" --project "$PROJECT"
  for sa in "$BACKEND" "$FRONTEND"; do
    gcloud iam service-accounts describe "${sa}@${PROJECT}.iam.gserviceaccount.com" --project "$PROJECT" >/dev/null 2>&1 ||
      gcloud iam service-accounts create "$sa" --project "$PROJECT"
  done
  # backend は Vertex AI（Gemini）と Firestore だけ使える。frontend には権限を付けない
  for role in roles/aiplatform.user roles/datastore.user; do
    gcloud projects add-iam-policy-binding "$PROJECT" --member "serviceAccount:${BACKEND_SA}" --role "$role" --condition None >/dev/null
  done
  gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet
}

deploy() {
  docker build -t "${AR}/backend:latest" backend
  docker build -t "${AR}/frontend:latest" \
    --build-arg NEXT_PUBLIC_API_URL="$BACKEND_URL" \
    --build-arg NEXT_PUBLIC_GOOGLE_CLIENT_ID="$(env_get GOOGLE_CLIENT_ID)" \
    frontend
  docker push "${AR}/backend:latest"
  docker push "${AR}/frontend:latest"

  # カンマを含む値（ADMIN_EMAILS など）があるので、環境変数はファイルで渡す
  local envfile
  envfile="$(mktemp)"
  trap 'rm -f "$envfile"' RETURN
  cat >"$envfile" <<YAML
STORAGE: "firestore"
GOOGLE_CLOUD_PROJECT: "${PROJECT}"
GOOGLE_CLOUD_LOCATION: "$(env_get GOOGLE_CLOUD_LOCATION)"
TUNED_MODEL_TSUKUYOMI: "$(env_get TUNED_MODEL_TSUKUYOMI)"
GOOGLE_CLIENT_ID: "$(env_get GOOGLE_CLIENT_ID)"
ADMIN_EMAILS: "$(env_get ADMIN_EMAILS)"
ALLOWED_ORIGIN: "${FRONTEND_URL}"
TRUST_PROXY: "true"
TZ: "Asia/Tokyo"
YAML
  gcloud run deploy "$BACKEND" --image "${AR}/backend:latest" --region "$REGION" --project "$PROJECT" \
    --service-account "$BACKEND_SA" --env-vars-file "$envfile" \
    --allow-unauthenticated --max-instances 2 --quiet
  gcloud run deploy "$FRONTEND" --image "${AR}/frontend:latest" --region "$REGION" --project "$PROJECT" \
    --service-account "$FRONTEND_SA" --set-env-vars "BACKEND_URL=${BACKEND_URL},COOKIE_SECURE=true" \
    --allow-unauthenticated --max-instances 2 --quiet

  echo
  echo "frontend: ${FRONTEND_URL}"
  # /healthz は Cloud Run が予約しているパスなので届かない。動作確認はキャラ一覧で行う
  echo "backend:  ${BACKEND_URL}/api/characters"
  echo "管理画面の Google ログインには、OAuth クライアントの「承認済みの JavaScript 生成元」に ${FRONTEND_URL} の追加が必要"
}

teardown() {
  gcloud run services delete "$FRONTEND" --region "$REGION" --project "$PROJECT" --quiet || true
  gcloud run services delete "$BACKEND" --region "$REGION" --project "$PROJECT" --quiet || true
  gcloud artifacts repositories delete "$REPO" --location "$REGION" --project "$PROJECT" --quiet || true
  for role in roles/aiplatform.user roles/datastore.user; do
    gcloud projects remove-iam-policy-binding "$PROJECT" --member "serviceAccount:${BACKEND_SA}" --role "$role" --condition None >/dev/null || true
  done
  gcloud iam service-accounts delete "$BACKEND_SA" --project "$PROJECT" --quiet || true
  gcloud iam service-accounts delete "$FRONTEND_SA" --project "$PROJECT" --quiet || true
  echo "片付けました（Firestore のデータとチューニング済みモデルは残しています）"
}

case "${1:-}" in
  setup) setup ;;
  deploy) deploy ;;
  teardown) teardown ;;
  *) sed -n '2,8p' "$0"; exit 1 ;;
esac
