#!/bin/sh
#
# This script is used to compile your program

set -e # Exit on failure

go build -o /tmp/build-shell-go app/*.go
