#!/bin/bash
if [ ! -f ./dist/tidyyy ]; then
    echo "Binary not found. Running build..."
    ./build.sh
fi

./dist/tidyyy "$@"
