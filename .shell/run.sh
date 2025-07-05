#!/bin/sh
#
# This script is used to run your program
#
# This runs after .shell/compile.sh

set -e # Exit on failure

exec /tmp/build-shell-go "$@"
