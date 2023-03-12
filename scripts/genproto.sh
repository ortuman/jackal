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
  "model/v1/archive.proto"
  "model/v1/user.proto"
  "model/v1/last.proto"
  "model/v1/blocklist.proto"
  "model/v1/caps.proto"
  "model/v1/roster.proto"
)

# Create the vendor/ dir for protobuf files to import from, and clean it up at
# the end of the script (regardless of the exit status).
# We move the vendor directory to a new name to avoid "go run" trying to use it.
function cleanup {
	rm -rf ./tmp_vendor
}
trap cleanup EXIT
go mod vendor -o tmp_vendor

for file in "${FILES[@]}"; do
  PATH="$PWD/scripts:$PATH" protoc \
    --proto_path=./tmp_vendor \
    --proto_path=. \
    --go_out=. \
    --go-grpc_out=. \
    proto/"${file}"
done
