#!/usr/bin/env bash
set -eufo pipefail

command -v helm >/dev/null 2>&1 || { echo "helm not installed, aborting." >&2; exit 1; }

if [ "$#" -eq 0 ] || [ -z "$1" ]; then
   echo "A custom values.yaml file must be provided"
   exit 1;
fi

helm install jackal helm/ --dependency-update --create-namespace --namespace=jackal -f "$1"

