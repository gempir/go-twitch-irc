#!/bin/bash

SUCCESS = 0

while true; do
    reset
    if ! make test; then
        echo "Failed after ${SUCCESS} successful runs"
        exit 1
    fi
    SUCCESS=$((SUCCESS + 1))
done
