#! /usr/bin/env nix-shell
#! nix-shell ../build.nix -i bash

set -o errexit
set -o nounset
set -o pipefail

go build -a -ldflags="-v -extldflags '-static'" -tags "netgo sqlite_omit_load_extension sqlite_json1 sqlite_icu"

# ssh -t deployer@pandora.stravid.com -p 5020 "sudo systemctl stop besserliste"
# scp -P 5020 besserliste deployer@pandora.stravid.com:~/apps/besserliste/
# ssh -t deployer@pandora.stravid.com -p 5020 "sudo systemctl restart besserliste"
