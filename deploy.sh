#!/usr/bin/env bash
CRUX="sdsheetal:latest"
IMAGE="sdsheetal:latest"
VOL="/data"
HOST="${HOST:-62.238.18.85}"
# quick deploy from this repo on the target host
if [ -d /opt/sdsheetal ]; then
  cd /opt/sdsheetal
elif command -v git; then
  git clone "https://github.com/vishalgojha/sd.git" /opt/sdsheetal; cd /opt/sdsheetal
else
  echo "no git, failing"
  exit 1
fi
docker buildx build . -t "$IMAGE"
docker stop sdsheetal 2>/dev/null; docker rm sdsheetal 2>/dev/null
docker run -d --name sdsheetal --restart unless-stopped -v "$VOL:/data" -p 8080:8080 "$IMAGE"
curl -s "https://sd.vishalojha.me/healthz"
