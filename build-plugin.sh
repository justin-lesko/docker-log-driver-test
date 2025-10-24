#!/usr/bin/env bash
set -euo pipefail

# --- Usage & inputs -----------------------------------------------------------
if [[ "${1:-}" == "" ]]; then
  echo "Usage: $0 <PLUGIN_NAME> [--no-cache] [--image-tag <tag>] [--dockerfile <path>]"
  echo "  PLUGIN_NAME example: myorg/docker-logging-plugin:latest"
  exit 1
fi

PLUGIN_NAME="$1"; shift || true

# Defaults (can be overridden via flags or env)
DOCKERFILE="${DOCKERFILE:-Dockerfile.build}"
IMAGE_TAG="${IMAGE_TAG:-docker-logging-plugin}"
NO_CACHE=false

# Parse flags
while [[ "${1:-}" != "" ]]; do
  case "$1" in
    --no-cache) NO_CACHE=true ;;
    --image-tag) IMAGE_TAG="$2"; shift ;;
    --dockerfile) DOCKERFILE="$2"; shift ;;
    *) echo "Unknown option: $1" && exit 2 ;;
  esac
  shift || true
done

# --- Pre-flight checks --------------------------------------------------------
command -v docker >/dev/null 2>&1 || { echo "Docker is required."; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "tar is required."; exit 1; }

[[ -f "$DOCKERFILE" ]] || { echo "Missing $DOCKERFILE"; exit 1; }
[[ -f "config.json" ]] || { echo "Missing config.json in current directory"; exit 1; }

# Ensure Docker is reachable
docker info >/dev/null 2>&1 || { echo "Docker daemon not reachable."; exit 1; }

# --- Paths & names ------------------------------------------------------------
PLUGIN_DIR="./plugin"
ROOTFS_DIR="$PLUGIN_DIR/rootfs"
TMP_CTN="tmp-plugin-$(date +%s)"
BUILD_ARGS=()
$NO_CACHE && BUILD_ARGS+=(--no-cache)

# Cleanup handler
cleanup() {
  set +e
  docker rm -f "$TMP_CTN" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# --- Build image --------------------------------------------------------------
echo "==> Building image '$IMAGE_TAG' from '$DOCKERFILE'..."
docker build -f "$DOCKERFILE" -t "$IMAGE_TAG" "${BUILD_ARGS[@]}" .

# --- Create temp container ----------------------------------------------------
echo "==> Creating temporary container '$TMP_CTN'..."
docker container create --name "$TMP_CTN" "$IMAGE_TAG" >/dev/null

# --- Prepare plugin directory -------------------------------------------------
echo "==> Preparing plugin directory at '$PLUGIN_DIR'..."
mkdir -p "$ROOTFS_DIR"
# Clean existing rootfs to avoid stale files
if [[ -n "$(ls -A "$ROOTFS_DIR" 2>/dev/null || true)" ]]; then
  rm -rf "$ROOTFS_DIR"/*
fi

echo "==> Copying config.json..."
cp -f config.json "$PLUGIN_DIR/"

# --- Export container filesystem into rootfs ---------------------------------
echo "==> Exporting container filesystem to '$ROOTFS_DIR'..."
docker container export "$TMP_CTN" | tar -x -C "$ROOTFS_DIR"

# --- (Optional) remove existing plugin with same name -------------------------
if docker plugin inspect "$PLUGIN_NAME" >/dev/null 2>&1; then
  echo "==> Plugin '$PLUGIN_NAME' already exists. Disabling & removing it..."
  docker plugin disable -f "$PLUGIN_NAME" >/dev/null 2>&1 || true
  docker plugin rm -f "$PLUGIN_NAME" >/dev/null
fi

# --- Create the Docker plugin -------------------------------------------------
echo "==> Creating plugin '$PLUGIN_NAME' from './plugin'..."
docker plugin create "$PLUGIN_NAME" "$PLUGIN_DIR"

echo "==> Done!"
echo "Created plugin: $PLUGIN_NAME"
echo
echo "Next steps:"
echo "  docker plugin ls | grep \"$(echo "$PLUGIN_NAME" | cut -d: -f1)\""
echo "  docker plugin enable $PLUGIN_NAME"

