#!/usr/bin/env bash
# Vertex AI で Gemini の教師ありチューニングを行うスクリプト。
#
#   scripts/tuning.sh upload   # data/processed の JSONL を Cloud Storage に上げる
#   scripts/tuning.sh create   # チューニングジョブを作る（料金がかかる）
#   scripts/tuning.sh status [JOB]  # ジョブの状態を見る（JOB を省くと一覧）
#
# 前提: gcloud auth login 済みで、.env に GOOGLE_CLOUD_PROJECT がある。
# Gemini 3.x のチューニングは us / eu のマルチリージョンでしか動かない。
set -euo pipefail
cd "$(dirname "$0")/.."

PROJECT="${GOOGLE_CLOUD_PROJECT:-$(grep '^GOOGLE_CLOUD_PROJECT=' .env | cut -d= -f2-)}"
LOCATION="${TUNING_LOCATION:-us}"
BASE_MODEL="${BASE_MODEL:-gemini-3.5-flash}"
DATASET="${DATASET:-tsukuyomi/v1}"
DISPLAY_NAME="${DISPLAY_NAME:-tsukuyomi-v1}"
BUCKET="gs://${PROJECT}-tuning"
API="https://aiplatform.${LOCATION}.rep.googleapis.com/v1/projects/${PROJECT}/locations/${LOCATION}/tuningJobs"

call() {
  curl -sS -H "Authorization: Bearer $(gcloud auth print-access-token)" \
    -H "Content-Type: application/json" "$@"
}

case "${1:-}" in
  upload)
    gcloud storage cp data/processed/train.jsonl data/processed/validation.jsonl "${BUCKET}/${DATASET}/"
    ;;
  create)
    # ハイパーパラメータは推奨値（未指定）のまま。まずはこれで1回試す
    call -X POST "$API" -d @- <<EOF
{
  "baseModel": "${BASE_MODEL}",
  "supervisedTuningSpec": {
    "trainingDatasetUri": "${BUCKET}/${DATASET}/train.jsonl",
    "validationDatasetUri": "${BUCKET}/${DATASET}/validation.jsonl"
  },
  "tunedModelDisplayName": "${DISPLAY_NAME}"
}
EOF
    echo
    ;;
  status)
    if [[ -n "${2:-}" ]]; then
      call "${API}/${2##*/}"
    else
      call "$API"
    fi
    echo
    ;;
  *)
    sed -n '2,10p' "$0"
    exit 1
    ;;
esac
