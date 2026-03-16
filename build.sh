#!/bin/bash

mkdir -p dist

go build -o ./dist/tidyyy .
echo "Build successful: ./dist/tidyyy"