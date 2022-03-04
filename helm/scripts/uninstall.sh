#!/usr/bin/env bash
set -eufo pipefail

command -v helm >/dev/null 2>&1 || { echo "helm not installed, aborting." >&2; exit 1; }

helm uninstall jackal --namespace=jackal
