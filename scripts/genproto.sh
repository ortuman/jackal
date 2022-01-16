#!/usr/bin/env bash
set -eufo pipefail

command -v protoc >/dev/null 2>&1 || { echo "protoc not installed,  Aborting." >&2; exit 1; }

if ! [[ "$0" =~ scripts/genproto.sh ]]; then
	echo "Must be run from repository root"
	exit 255
fi

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

FILES=( "admin/v1/users.proto" "c2s/v1/resourceinfo.proto" "cluster/v1/cluster.proto" )

for file in "${FILES[@]}"; do
  protoc \
    --proto_path=${GOPATH}/src \
    --proto_path=. \
    --go_out=. \
    --go-grpc_out=. \
    proto/"${file}"
done
