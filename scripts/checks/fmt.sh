set -eufo pipefail

command -v goimports >/dev/null 2>&1 || { echo 'Please install goimports or use image that has it'; exit 1; }

find . -name '*.go' -exec goimports -l {} +
