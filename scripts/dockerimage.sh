#!/bin/bash
set -eufo pipefail

command -v go >/dev/null 2>&1 || { echo 'Please install go or use image that has it'; exit 1; }
command -v docker >/dev/null 2>&1 || { echo 'Please install docker or use image that has it'; exit 1; }

rm -rf build/
mkdir build

for arch in amd64 arm64
do
  CGO_ENABLED=0 GOOS=linux GOARCH=${arch} go build -a -tags netgo \
    -ldflags "-s -w" -o "build/${arch}/jackal" "github.com/ortuman/jackal/cmd/jackal"
  CGO_ENABLED=0 GOOS=linux GOARCH=${arch} go build -a -tags netgo \
    -ldflags "-s -w" -o "build/${arch}/jackalctl" "github.com/ortuman/jackal/cmd/jackalctl"

  docker buildx build --platform linux/${arch} -f dockerfiles/Dockerfile .
done

docker buildx build --platform linux/amd64,linux/arm64 -t ortuman/jackal -o type=registry -f dockerfiles/Dockerfile .

