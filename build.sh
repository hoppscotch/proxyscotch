#!/bin/bash

if [ $# -lt 1 ]; then
  echo "Usage: $0 <darwin|linux|windows>"
  exit 3
fi

# Collect parameters.
PLATFORM="$1"

# Remove all legacy icons.
[ -f icons/icon_unix.go ] && rm icons/icon_unix.go
[ -f icons/icon_win.go ] && rm icons/icon_win.go

# Build the icon for the appropriate platform.
if [ "$PLATFORM" == "darwin" ] || [ "$PLATFORM" == "linux" ]; then
  cat "icons/icon.png" | go run github.com/cratonica/2goarray Data icon >> icons/icon_unix.go
elif [ "$PLATFORM" == "windows" ]; then
  cat "icons/icon.ico" | go run github.com/cratonica/2goarray Data icon >> icons/icon_win.go
else
  echo "Unknown platform: $1"
  exit 3
fi

[ -d "out/" ] || mkdir "out/"
[ -d "out/$PLATFORM" ] && rm -r "out/$PLATFORM"
mkdir "out/$PLATFORM"
cp -r "resources/$PLATFORM" "out"

if [ "$PLATFORM" == "darwin" ]; then
  mkdir -p "out/darwin/PostwomanProxy.app/Contents/MacOS"
  mkdir -p "out/darwin/PostwomanProxy.app/Contents/MacOS/icons"
  cp icons/icon.png out/darwin/PostwomanProxy.app/Contents/MacOS/icons/
  GOOS="darwin" go build -o "out/darwin/PostwomanProxy.app/Contents/MacOS/postwoman-proxy"
elif [ "$PLATFORM" == "windows" ]; then
  [ -f "rsrc.syso" ] && rm rsrc.syso
  go get github.com/akavel/rsrc

  rsrc -manifest="out/windows/postwoman-proxy.manifest" -ico="icons/icon.ico" -o rsrc.syso
  GOOS="windows" go build -ldflags -H=windowsgui -o "out/windows/postwoman-proxy.exe"

  mkdir out/windows/icons
  cp icons/icon.png "out/windows/icons/icon.png"

  mkdir out/windows/data

  rm out/windows/postwoman-proxy.manifest
  rm rsrc.syso
elif [ "$PLATFORM" == "linux" ]; then
  echo "NOTICE: postwoman-proxy is untested and currently unsupported on Linux."
  GOOS="linux" go build -o "out/linux/postwoman"
fi
