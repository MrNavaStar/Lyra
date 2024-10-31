#!/bin/bash

platforms=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for platform in "${platforms[@]}"; do
  IFS="/" read -r GOOS GOARCH <<< "$platform"
  output="bin/lyra-$GOOS-$GOARCH"
  [[ "$GOOS" == "windows" ]] && output+=".exe"

  echo "Building for $GOOS/$GOARCH..."
  env GOOS=$GOOS GOARCH=$GOARCH go build -o "$output" -tags "llua lua54"
done
