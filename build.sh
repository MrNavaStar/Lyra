#!/bin/bash

platforms=(
  "linux/amd64"
  "windows/amd64"
  "darwin/amd64"
  "linux/arm64"
  "windows/arm64"
  "darwin/arm64"
)

for platform in "${platforms[@]}"; do
  IFS="/" read -r GOOS GOARCH <<< "$platform"
  output="bin/app-$GOOS-$GOARCH"
  [[ "$GOOS" == "windows" ]] && output+=".exe"

  echo "Building for $GOOS/$GOARCH..."
  env GOOS=$GOOS GOARCH=$GOARCH go build -o "$output"
done

