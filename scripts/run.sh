#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

watchexec -r -w main.go -w undo.go -w set_quantity.go -w web -w types -w queries --shell=none -- go run -tags "sqlite_omit_load_extension sqlite_json1 sqlite_icu" *.go
