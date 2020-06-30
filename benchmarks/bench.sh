#!/bin/bash

set -e

cd $(dirname "${BASH_SOURCE[0]}")

if [ ! -d "results/" ];then
    mkdir -p results/
fi

./bench-echo.sh 2>&1 | tee results/echo.txt

go run analyze.go