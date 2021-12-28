#!/bin/bash
set -eufo pipefail

command -v go >/dev/null 2>&1 || { echo 'Please install go or use image that has it'; exit 1; }

go test -timeout 1m --race -coverprofile=coverage.txt -covermode=atomic ./...
