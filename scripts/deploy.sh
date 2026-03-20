#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

rm -f ./besserliste
go build -tags "icu"

scp ./besserliste deployer@app001.stravid.com:/srv/besserliste/besserliste.tmp

ssh -t deployer@app001.stravid.com << EOF
  sudo systemctl stop besserliste
  rm -f /srv/besserliste/besserliste
  mv /srv/besserliste/besserliste.tmp /srv/besserliste/besserliste
  sudo systemctl start besserliste
EOF

rm -f ./besserliste
