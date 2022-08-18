#!/usr/bin/env bash
set -eufo pipefail

command -v protoc >/dev/null 2>&1 || { echo "protoc not installed,  Aborting." >&2; exit 1; }

if ! [[ "$0" =~ scripts/genproto.sh ]]; then
	echo "Must be run from repository root"
	exit 255
fi

FILES=(
  "admin/v1/users.proto"
  "c2s/v1/resourceinfo.proto"
  "cluster/v1/cluster.proto"
  "model/v1/user.proto"
  "model/v1/last.proto"
  "model/v1/blocklist.proto"
  "model/v1/caps.proto"
  "model/v1/roster.proto"
)

for file in "${FILES[@]}"; do
  PATH="$(pwd)/scripts:$PATH" protoc \
    --proto_path=${GOPATH}/src \
    --proto_path=. \
    --go_out=. \
    --go-grpc_out=. \
    proto/"${file}"
done
