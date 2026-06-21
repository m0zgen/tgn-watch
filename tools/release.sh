#!/usr/bin/env bash
set -euo pipefail

APP_NAME="tgn-watch"
VERSION="${1:-}"

if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 v0.1.1"
  exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid version: $VERSION"
  echo "Expected format: v0.1.1"
  exit 1
fi

if [[ -n "$(git status --short)" ]]; then
  echo "Git working tree is dirty. Commit changes first:"
  git status --short
  exit 1
fi

echo "[release] fetching tags"
git fetch --tags --quiet

if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "Tag already exists locally: $VERSION"
  exit 1
fi

if git ls-remote --tags origin "refs/tags/$VERSION" | grep -q "$VERSION"; then
  echo "Tag already exists on origin: $VERSION"
  exit 1
fi

LATEST_TAG="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n 1 || true)"

if [[ -n "$LATEST_TAG" ]]; then
  LOWEST="$(printf '%s\n%s\n' "$LATEST_TAG" "$VERSION" | sort -V | head -n 1)"

  if [[ "$LOWEST" == "$VERSION" ]]; then
    echo "Version must be greater than latest tag."
    echo "Latest tag: $LATEST_TAG"
    echo "Requested:   $VERSION"
    exit 1
  fi
fi

echo "[release] running tests"
go test ./...

echo "[release] checking GoReleaser config"
goreleaser check

echo "[release] creating tag $VERSION"
git tag -a "$VERSION" -m "$APP_NAME $VERSION"

echo "[release] pushing tag $VERSION"
git push origin "$VERSION"

echo "[release] done"
echo "GitHub Actions should now build and publish the release."
