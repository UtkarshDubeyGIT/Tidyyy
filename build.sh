#!/bin/bash

mkdir -p dist

go build -o ./dist/tidyyy ./cmd/tidyyy
echo "Build successful: ./dist/tidyyy"