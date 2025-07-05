#!/bin/sh
#
# Use this script to run your program LOCALLY.

set -e # Exit early if any commands fail

# - Edit this to change how your program compiles locally
# - Edit .shell/compile.sh to change how your program compiles remotely
(
  cd "$(dirname "$0")" # Ensure compile steps are run within the repository directory
  go build -o /tmp/build-shell-go app/*.go
)


# - Edit this to change how your program runs locally
# - Edit .shell/run.sh to change how your program runs remotely
exec /tmp/build-shell-go "$@"
