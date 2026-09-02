#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "${script_dir}/common.sh"

require_command go

printf '%s\n' '==> Testing the Go application'
(
    cd "${release_root_dir}"
    go test -race ./...
    go vet ./...
)
