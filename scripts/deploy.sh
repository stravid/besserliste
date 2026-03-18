#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

rm ./besserliste
go build -tags "icu"

scp ./besserliste deployer@app001.stravid.com:/srv/besserliste/besserliste.tmp

ssh -t deployer@pandora.stravid.com -p 5020 << EOF
  systemctl stop besserliste
  rm /srv/besserliste/besserliste
  mv /srv/besserliste/besserliste.tmp /srv/besserliste/besserliste
  systemctl start besserliste
EOF
