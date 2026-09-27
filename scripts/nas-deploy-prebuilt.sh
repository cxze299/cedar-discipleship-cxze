#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GIT="${GIT:-/usr/local/bin/git}"
DOCKER="${DOCKER:-/usr/local/bin/docker}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deploy/docker-compose.separated.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
LOCK_DIR="${AGP_DEPLOY_LOCK_DIR:-/tmp/cedar-prebuilt-deploy.lock}"

log() {
  printf '[%s] %s\n' "$(date '+%F %T')" "$*"
}

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

docker_cmd() {
  if [ "${AGP_DOCKER_USE_SUDO:-true}" = "true" ]; then
    sudo -n "$DOCKER" "$@"
  else
    "$DOCKER" "$@"
  fi
}

lower() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

github_path_from_origin() {
  local origin path
  origin="$("$GIT" -C "$ROOT_DIR" remote get-url origin 2>/dev/null || true)"
  case "$origin" in
    git@github.com:*)
      path="${origin#git@github.com:}"
      ;;
    https://github.com/*)
      path="${origin#https://github.com/}"
      ;;
    *)
      path="wangz5940/cedar-discipleship"
      ;;
  esac
  path="${path%.git}"
  printf '%s\n' "$path"
}

if [ ! -f "$ENV_FILE" ]; then
  fail "missing env file: $ENV_FILE"
fi

if ! mkdir "$LOCK_DIR" 2>/dev/null; then
  fail "another deployment appears to be running: $LOCK_DIR"
fi

IMAGE_ENV=""
IMAGE_COMPOSE=""
cleanup() {
  if [ -n "$IMAGE_ENV" ] && [ -f "$IMAGE_ENV" ]; then
    rm -f "$IMAGE_ENV"
  fi
  if [ -n "$IMAGE_COMPOSE" ] && [ -f "$IMAGE_COMPOSE" ]; then
    rm -f "$IMAGE_COMPOSE"
  fi
  rmdir "$LOCK_DIR" 2>/dev/null || true
}
trap cleanup EXIT HUP INT TERM

cd "$ROOT_DIR"

if [ -n "$("$GIT" status --porcelain=v1 -uno)" ]; then
  fail "tracked worktree changes exist on NAS; stash or commit them before deploying"
fi

log "Syncing repository"
"$GIT" fetch origin --prune
"$GIT" fetch origin master
deploy_commit="$("$GIT" rev-parse --verify "${AGP_GIT_REF:-origin/master}^{commit}")"
"$GIT" merge-base --is-ancestor "$deploy_commit" origin/master ||
  fail "deployment commit is not included in origin/master: $deploy_commit"
"$GIT" checkout --detach "$deploy_commit"

set +u
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a
set -u

COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cedar}"
AGP_CONTAINER_PREFIX="${AGP_CONTAINER_PREFIX:-cedar}"
AGP_IMAGE_REGISTRY="${AGP_IMAGE_REGISTRY:-ghcr.io}"
AGP_IMAGE_TAG="${AGP_IMAGE_TAG:-$deploy_commit}"

github_path="$(lower "$(github_path_from_origin)")"
AGP_IMAGE_OWNER="${AGP_IMAGE_OWNER:-${github_path%%/*}}"
AGP_IMAGE_REPO="${AGP_IMAGE_REPO:-${github_path#*/}}"

AGP_BACKEND_IMAGE="${AGP_BACKEND_IMAGE:-${AGP_IMAGE_REGISTRY}/${AGP_IMAGE_OWNER}/${AGP_IMAGE_REPO}-backend:${AGP_IMAGE_TAG}}"
AGP_FRONTEND_IMAGE="${AGP_FRONTEND_IMAGE:-${AGP_IMAGE_REGISTRY}/${AGP_IMAGE_OWNER}/${AGP_IMAGE_REPO}-frontend:${AGP_IMAGE_TAG}}"

IMAGE_ENV="$(mktemp /tmp/cedar-images.XXXXXX)"
{
  printf 'AGP_BACKEND_IMAGE=%s\n' "$AGP_BACKEND_IMAGE"
  printf 'AGP_FRONTEND_IMAGE=%s\n' "$AGP_FRONTEND_IMAGE"
} >"$IMAGE_ENV"
chmod 600 "$IMAGE_ENV"

compose_args=(
  --env-file "$ENV_FILE"
  --env-file "$IMAGE_ENV"
  -p "$COMPOSE_PROJECT_NAME"
  -f "$COMPOSE_FILE"
)

log "Pulling prebuilt images for commit $deploy_commit (tag $AGP_IMAGE_TAG)"
if ! docker_cmd compose "${compose_args[@]}" pull backend frontend; then
  cat >&2 <<EOF

Unable to pull prebuilt images.
If the GHCR packages are private, log in once on the NAS:
  sudo /usr/local/bin/docker login ghcr.io -u <github-user>

Expected images:
  $AGP_BACKEND_IMAGE
  $AGP_FRONTEND_IMAGE
EOF
  exit 1
fi

verified_image_id() {
  local image="$1" image_id revision
  image_id="$(docker_cmd image inspect --format '{{.Id}}' "$image")"
  [[ "$image_id" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "invalid image ID: $image"
  revision="$(docker_cmd image inspect --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' "$image_id")"
  [ "$revision" = "$deploy_commit" ] ||
    fail "image revision does not match commit $deploy_commit: $image"
  printf '%s\n' "$image_id"
}

backend_image_id="$(verified_image_id "$AGP_BACKEND_IMAGE")"
frontend_image_id="$(verified_image_id "$AGP_FRONTEND_IMAGE")"
# Pin the exact images inspected above, including when a custom Compose file is used.
IMAGE_COMPOSE="$(mktemp /tmp/cedar-images-compose.XXXXXX)"
{
  printf 'services:\n  backend:\n    image: "%s"\n  frontend:\n    image: "%s"\n' \
    "$backend_image_id" "$frontend_image_id"
} >"$IMAGE_COMPOSE"
compose_args+=(-f "$IMAGE_COMPOSE")
log "Verified commit=$deploy_commit backend=$backend_image_id frontend=$frontend_image_id"

log "Starting containers without rebuilding"
docker_cmd compose "${compose_args[@]}" up -d --no-build --pull never backend frontend

log "Current containers"
docker_cmd ps --filter "name=${AGP_CONTAINER_PREFIX}-" --format '{{.Names}} {{.Status}} {{.Image}}'
