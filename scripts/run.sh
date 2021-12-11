#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

watchexec -r -w main.go -w web go run main.go
