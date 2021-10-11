#!/bin/bash
set -eufo pipefail

command -v go >/dev/null 2>&1 || { echo 'Please install go or use image that has it'; exit 1; }
command -v docker >/dev/null 2>&1 || { echo 'Please install docker or use image that has it'; exit 1; }

rm -rf build/
mkdir build

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo \
  -ldflags "-s -w" -o "build/jackal" "github.com/ortuman/jackal/cmd/jackal"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo \
  -ldflags "-s -w" -o "build/jackalctl" "github.com/ortuman/jackal/cmd/jackalctl"
docker build -f dockerfiles/Dockerfile -t ortuman/jackal .
