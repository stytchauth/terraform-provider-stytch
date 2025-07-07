#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "You must enter exactly 1 command line argument: DEPENDENCY"
  exit 1
fi

DEPENDENCY="$1"

# Extract base path without the /vN suffix if present.
REPLACEMENT_PATH="$(echo "$DEPENDENCY" | sed -E 's#/v[0-9]+$##')"

LINE="replace github.com/stytchauth/$DEPENDENCY => ../$REPLACEMENT_PATH"

if grep -qF "$LINE" go.mod; then
  grep -vF "$LINE" go.mod > go.mod.tmp
  mv go.mod.tmp go.mod
else
  echo "$LINE" >> go.mod
fi