#!/bin/bash
set -eufo pipefail

command -v go >/dev/null 2>&1 || { echo 'Please install go or use image that has it'; exit 1; }

CGO_ENABLED=0 go install -ldflags "-s -w" "github.com/ortuman/jackal/pkg/cmd/jackal"
