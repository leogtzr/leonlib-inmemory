#!/bin/bash

export LEONLIB_DB_PASSWORD=${LEONLIB_DB_PASSWORD}
export LEONLIB_DB_USER="leo"
export LEONLIB_DB="leonlib"
export LEONLIB_DB_HOST="leonlib"
# sqlite, postgres or memory
export DB_MODE="sqlite"
export PORT=8180
export RUN_MODE="prod"

docker-compose -f docker-compose.sqlite.yml up --build

exit
