#!/bin/bash
set -eufo pipefail

command -v go >/dev/null 2>&1 || { echo 'Please install go or use image that has it'; exit 1; }
command -v docker >/dev/null 2>&1 || { echo 'Please install docker or use image that has it'; exit 1; }

for arch in amd64 arm64
do
  docker buildx build --platform linux/${arch} -f dockerfiles/Dockerfile .
done

docker buildx build --platform linux/amd64,linux/arm64 -t ortuman/jackal -o type=registry -f dockerfiles/Dockerfile .
