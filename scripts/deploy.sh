#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ssh -t deployer@pandora.stravid.com -p 5020 << EOF
  cd ~/code/besserliste
  git checkout master
  git pull
  export PATH=$PATH:/usr/local/go/bin
  go build -a -ldflags="-v -extldflags ''" -tags "netgo sqlite_omit_load_extension sqlite_json1 sqlite_icu"
  sudo systemctl stop besserliste
  cp besserliste ~/apps/besserliste/
  sudo systemctl restart besserliste
EOF
