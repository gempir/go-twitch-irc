#!/bin/bash

set -e

COUNT=1

while true; do
    tput reset
    echo $COUNT
    COUNT=$((COUNT+1))
    go test -race
    # make cover
    # go test -race -v -run TestPinger
    # sleep 3
    # go test -race -run TestCanNotUseImproperlyFormattedOauth
done
