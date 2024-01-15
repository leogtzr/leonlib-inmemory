#!/bin/bash

export LEONLIB_DB_PASSWORD=${LEONLIB_DB_PASSWORD}
export LEONLIB_DB_USER="leo"
export LEONLIB_DB="leonlib"
export LEONLIB_DB_HOST="leonlib"
# inmemory (sqlite) or postgres
export DB_MODE="memory"
export PORT=8180

# docker-compose -f docker-compose.sqlite.yml up --build
make clean && make && ./leonlib

exit
