#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


export CGO_LDFLAGS="$(pkg-config --libs --static icu-i18n) -lstdc++"
CGO_ENABLED=1 go build -tags "icu" -ldflags '-extldflags "-static"'
scp -P 5020 ./besserliste deployer@pandora.stravid.com:~/apps/besserliste/besserliste.tmp

ssh -t deployer@pandora.stravid.com -p 5020 << EOF
  sudo systemctl stop besserliste
  rm ~/apps/besserliste/besserliste
  mv ~/apps/besserliste/besserliste.tmp ~/apps/besserliste/besserliste
  sudo systemctl restart besserliste
EOF
