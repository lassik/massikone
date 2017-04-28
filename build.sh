#!/bin/sh
set -eu
cd "$(dirname "$0")"
mkdir -p data
sqlite3 data/massikone.sqlite <schema.sql
rm -f massikone-docker.tgz
git ls-files | xargs tar -czf massikone-docker.tgz
docker-compose up --build
