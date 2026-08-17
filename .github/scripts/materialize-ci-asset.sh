#!/usr/bin/env bash
# Materialize a CI asset path from the ci-assets branch (or pass through http(s) URLs).
# Usage: materialize-ci-asset.sh <path-or-url> <branch> <output-path>
# Prints the usable local path or URL on stdout.
set -euo pipefail

SRC="${1:-}"
BRANCH="${2:-ci-assets}"
OUT="${3:-.ci-fetched/asset}"

if [ -z "$SRC" ]; then
  echo ""
  exit 0
fi

# Already a remote URL — leave for the build tool to download.
if [[ "$SRC" == http://* || "$SRC" == https://* ]]; then
  printf '%s\n' "$SRC"
  exit 0
fi

# Legacy: file already present on the checked-out default branch.
if [ -e "$SRC" ]; then
  printf '%s\n' "$SRC"
  exit 0
fi

if [ -z "${GH_TOKEN:-}${GITHUB_TOKEN:-}" ]; then
  echo "GH_TOKEN/GITHUB_TOKEN required to fetch $SRC from $BRANCH" >&2
  exit 1
fi
export GH_TOKEN="${GH_TOKEN:-$GITHUB_TOKEN}"

REPO="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY required}"
ENC_PATH="$(python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe="/"))' "$SRC")"

API_PATH="repos/${REPO}/contents/${ENC_PATH}?ref=${BRANCH}"
META="$(gh api "$API_PATH" 2>/dev/null || true)"
if [ -z "$META" ]; then
  echo "failed to fetch $SRC from branch $BRANCH" >&2
  exit 1
fi

TYPE="$(printf '%s' "$META" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("type","") if isinstance(d,dict) else "dir")')"

if [ "$TYPE" = "file" ]; then
  mkdir -p "$(dirname "$OUT")"
  printf '%s' "$META" | python3 -c 'import json,sys,base64; d=json.load(sys.stdin); open(sys.argv[1],"wb").write(base64.b64decode(d["content"]))' "$OUT"
  printf '%s\n' "$OUT"
  exit 0
fi

# Directory listing (inject scripts, etc.)
mkdir -p "$OUT"
printf '%s' "$META" | python3 -c '
import json,sys,os,base64,urllib.parse,subprocess
entries=json.load(sys.stdin)
out=sys.argv[1]
branch=sys.argv[2]
repo=os.environ["GITHUB_REPOSITORY"]
if not isinstance(entries, list):
    raise SystemExit("expected directory listing")
for e in entries:
    if e.get("type")!="file":
        continue
    path=e["path"]
    name=e["name"]
    enc=urllib.parse.quote(path, safe="/")
    raw=subprocess.check_output(["gh","api",f"repos/{repo}/contents/{enc}?ref={branch}"])
    meta=json.loads(raw)
    data=base64.b64decode(meta["content"])
    dest=os.path.join(out, name)
    open(dest,"wb").write(data)
' "$OUT" "$BRANCH"
printf '%s\n' "$OUT"
