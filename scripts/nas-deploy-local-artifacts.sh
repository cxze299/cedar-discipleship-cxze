#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SSH_HOST="${AGP_NAS_SSH_HOST:-yimaneili@mouss.synology.me}"
SSH_PORT="${AGP_NAS_SSH_PORT:-7}"
REMOTE_ROOT="${AGP_NAS_ROOT:-/volume2/docker/cedar-discipleship}"
DOCKER="${AGP_NAS_DOCKER:-/usr/local/bin/docker}"
LOCK_DIR="${AGP_DEPLOY_LOCK_DIR:-/tmp/cedar-prebuilt-deploy.lock}"
SHA="${AGP_GIT_REF:-$(git -C "$ROOT_DIR" rev-parse HEAD)}"
SHORT_SHA="${SHA:0:12}"
STAGE_DIR="$(mktemp -d "/tmp/cedar-local-artifacts.${SHORT_SHA}.XXXXXX")"
REMOTE_STAGE="/tmp/cedar-local-artifacts.${SHORT_SHA}.$$"

log() {
  printf '[%s] %s\n' "$(date '+%F %T')" "$*"
}

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

cleanup() {
  rm -rf "$STAGE_DIR"
  ssh -p "$SSH_PORT" "$SSH_HOST" "rm -rf '$REMOTE_STAGE'" >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

image_commit() {
  local image="$1"
  local tag="${image##*:}"
  if [[ "$tag" =~ ^[0-9a-f]{40}$ ]]; then
    printf '%s\n' "$tag"
  fi
}

verify_runtime_contract() {
  local base_image="$1"
  shift
  local base_commit
  base_commit="$(image_commit "$base_image")"
  if [ -z "$base_commit" ] || ! git -C "$ROOT_DIR" cat-file -e "$base_commit^{commit}" 2>/dev/null; then
    [ "${AGP_ALLOW_UNVERIFIED_BASE:-false}" = "true" ] ||
      fail "cannot verify deployed base image commit: $base_image"
    return
  fi
  git -C "$ROOT_DIR" diff --quiet "$base_commit..$SHA" -- "$@" ||
    fail "runtime image contract changed since $base_commit: $*"
}

require_command git
require_command go
require_command npm
require_command ssh
require_command tar

if [ -n "$(git -C "$ROOT_DIR" status --porcelain=v1 -uno)" ]; then
  fail "tracked worktree changes exist; commit them before deployment"
fi

log "Reading deployed image bases"
read -r BACKEND_BASE FRONTEND_BASE < <(
  ssh -p "$SSH_PORT" "$SSH_HOST" \
    "printf '%s %s\n' \
      \"\$(sudo -n '$DOCKER' inspect cedar-backend --format '{{.Config.Image}}')\" \
      \"\$(sudo -n '$DOCKER' inspect cedar-frontend --format '{{.Config.Image}}')\""
)
[ -n "$BACKEND_BASE" ] && [ -n "$FRONTEND_BASE" ] ||
  fail "unable to determine deployed image bases"

verify_runtime_contract "$BACKEND_BASE" backend/Dockerfile deploy/docker-compose.separated.yml
verify_runtime_contract "$FRONTEND_BASE" frontend/Dockerfile frontend/nginx.conf deploy/docker-compose.separated.yml

mkdir -p "$STAGE_DIR/backend/migrations" "$STAGE_DIR/frontend/dist"

log "Building Linux amd64 backend artifacts"
(
  cd "$ROOT_DIR/backend"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$STAGE_DIR/backend/agp-server" ./cmd/server
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$STAGE_DIR/backend/migrate-json" ./cmd/migrate-json
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o "$STAGE_DIR/backend/migrate-resource-files" ./cmd/migrate-resource-files
)
cp -R "$ROOT_DIR/backend/migrations/." "$STAGE_DIR/backend/migrations/"

log "Building frontend artifacts"
(
  cd "$ROOT_DIR/frontend"
  if [ ! -d node_modules ]; then
    npm ci --no-audit --no-fund
  fi
  npm run build
)
cp -R "$ROOT_DIR/frontend/dist/." "$STAGE_DIR/frontend/dist/"
cp "$ROOT_DIR/frontend/nginx.conf" "$STAGE_DIR/frontend/nginx.conf"

cat >"$STAGE_DIR/Dockerfile.backend" <<'EOF'
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
RUN rm -rf /app/migrations
COPY backend/migrations/ /app/migrations/
COPY backend/agp-server backend/migrate-json backend/migrate-resource-files /app/
EOF

cat >"$STAGE_DIR/Dockerfile.frontend" <<'EOF'
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
RUN rm -rf /usr/share/nginx/html/*
COPY frontend/dist/ /usr/share/nginx/html/
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
EOF

log "Transferring compressed artifacts to NAS"
ssh -p "$SSH_PORT" "$SSH_HOST" "mkdir -p '$REMOTE_STAGE'"
tar -C "$STAGE_DIR" -czf - . |
  ssh -p "$SSH_PORT" "$SSH_HOST" "tar -xzf - -C '$REMOTE_STAGE'"

BACKEND_IMAGE="local/cedar-discipleship-backend:$SHA"
FRONTEND_IMAGE="local/cedar-discipleship-frontend:$SHA"

log "Assembling and activating images on NAS"
ssh -p "$SSH_PORT" "$SSH_HOST" bash -s -- \
  "$REMOTE_ROOT" "$REMOTE_STAGE" "$DOCKER" "$LOCK_DIR" \
  "$BACKEND_BASE" "$FRONTEND_BASE" "$BACKEND_IMAGE" "$FRONTEND_IMAGE" <<'REMOTE'
set -euo pipefail

root_dir="$1"
stage_dir="$2"
docker="$3"
lock_dir="$4"
backend_base="$5"
frontend_base="$6"
backend_image="$7"
frontend_image="$8"
image_env=""

if ! mkdir "$lock_dir" 2>/dev/null; then
  echo "ERROR: deployment lock exists: $lock_dir" >&2
  exit 1
fi

cleanup() {
  rm -f "$image_env"
  rm -rf "$stage_dir"
  rmdir "$lock_dir" 2>/dev/null || true
}
trap cleanup EXIT HUP INT TERM

sudo -n "$docker" build \
  --build-arg "BASE_IMAGE=$backend_base" \
  -f "$stage_dir/Dockerfile.backend" \
  -t "$backend_image" \
  "$stage_dir"
sudo -n "$docker" build \
  --build-arg "BASE_IMAGE=$frontend_base" \
  -f "$stage_dir/Dockerfile.frontend" \
  -t "$frontend_image" \
  "$stage_dir"

image_env="$(mktemp /tmp/cedar-images.XXXXXX)"
{
  printf 'AGP_BACKEND_IMAGE=%s\n' "$backend_image"
  printf 'AGP_FRONTEND_IMAGE=%s\n' "$frontend_image"
} >"$image_env"
chmod 600 "$image_env"

cd "$root_dir"
sudo -n "$docker" compose \
  --env-file .env \
  --env-file "$image_env" \
  -p cedar \
  -f deploy/docker-compose.separated.yml \
  up -d --no-build backend frontend
sudo -n "$docker" ps \
  --filter name=cedar- \
  --format '{{.Names}} {{.Status}} {{.Image}}'
REMOTE

log "Deployment completed for $SHA"
