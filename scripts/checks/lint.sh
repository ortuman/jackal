set -eufo pipefail

command -v golint >/dev/null 2>&1 || { echo 'Please install goimports or use image that has it'; exit 1; }

golint -set_exit_status ./...
