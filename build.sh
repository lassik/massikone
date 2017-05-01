#!/bin/sh
set -eu
cd "$(dirname "$0")"
rm -f massikone-docker.tgz
git ls-files | xargs tar -czf massikone-docker.tgz
docker-compose up --build
