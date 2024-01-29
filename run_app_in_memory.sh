#!/bin/bash

export LEONLIB_DB_PASSWORD=${LEONLIB_DB_PASSWORD}
export LEONLIB_DB_USER="leo"
export LEONLIB_DB="leonlib"
export LEONLIB_DB_HOST="leonlib"
# sqlite, postgres or memory
export DB_MODE="memory"
export PORT=8180
export RUN_MODE="dev"
export LEONLIB_MAINAPP_USER=${LEONLIB_MAINAPP_USER}

make clean && make && ./leonlib

exit
