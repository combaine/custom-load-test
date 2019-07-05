#!/bin/bash
set -e

PLUGINS_PATH=./custom/ python3 ./custom.py &
background=$!

trap 'pkill -f "python3 ./custom.py"; rm -vf gun; exit 0' INT
GO111MODULE=on go build -o gun cmd/charge/main.go cmd/charge/main_gen.go
if test -e ./gun; then
    until sleep 1 && ./gun; do
        :
    done
fi
