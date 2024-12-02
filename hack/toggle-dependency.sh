#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 1 ]; then
	echo "You must enter exactly 1 command line arguments: DEPENDENCY"
	exit 1
fi

DEPENDENCY=$1
LINE="replace github.com/stytchauth/$DEPENDENCY => ../$DEPENDENCY"

if grep -q "$LINE" go.mod; then
	grep -v "$LINE" go.mod >go.mod.tmp
	mv go.mod.tmp go.mod
else
	echo $LINE >>go.mod
fi
