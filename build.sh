#!/bin/sh

if [ $# -lt 1 ]; then
  echo "Usage: $0 <darwin|linux|windows> [<server|desktop>]"
  exit 3
fi

# Collect parameters.
PLATFORM="$1"
BUILD_TYPE="$2"

# Ensure that a valid OS/platform has been selected.
if [ "$PLATFORM" != "darwin" ] && [ "$PLATFORM" != "linux" ] && [ "$PLATFORM" != "windows" ]; then
	echo "Invalid platform selected ($PLATFORM). It must be one of <darwin|linux|windows>."
	exit 4
fi

# Ensure that a valid build type has been selected.
if [ $# -lt 2 ]; then
	BUILD_TYPE="desktop"
elif [ "$BUILD_TYPE" != "desktop" ] && [ "$BUILD_TYPE" != "server" ]; then
	echo "Invalid build type selected ($BUILD_TYPE). It must be one of <server|desktop>."
	exit 5
fi

# Ensure output directory exists.
[ -d "out/" ] || mkdir "out/"
[ -d "out/$PLATFORM-$BUILD_TYPE" ] && rm -r "out/$PLATFORM-$BUILD_TYPE"
mkdir "out/$PLATFORM-$BUILD_TYPE"
OUTPUT_DIR="out/$PLATFORM-$BUILD_TYPE"

# Handle special build: server
if [ "$BUILD_TYPE" == "server" ]; then
	if [ "$PLATFORM" == "windows" ]; then
		GOOS="$PLATFORM" go build -o "$OUTPUT_DIR/postwoman-proxy-server.exe" server/server.go
	else
		GOOS="$PLATFORM" go build -o "$OUTPUT_DIR/postwoman-proxy-server" server/server.go
	fi
	exit
fi

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

cp -r "resources/$PLATFORM/." "$OUTPUT_DIR/"

if [ "$PLATFORM" == "darwin" ]; then
  mkdir -p "$OUTPUT_DIR/PostwomanProxy.app/Contents/MacOS"
  mkdir -p "$OUTPUT_DIR/PostwomanProxy.app/Contents/MacOS/icons"
  cp icons/icon.png "$OUTPUT_DIR/PostwomanProxy.app/Contents/MacOS/icons/"
  GOOS="darwin" go build -o "$OUTPUT_DIR/PostwomanProxy.app/Contents/MacOS/postwoman-proxy"
elif [ "$PLATFORM" == "windows" ]; then
  [ -f "rsrc.syso" ] && rm rsrc.syso
  go get github.com/akavel/rsrc

  rsrc -manifest="$OUTPUT_DIR/postwoman-proxy.manifest" -ico="icons/icon.ico" -o rsrc.syso
  GOOS="windows" go build -ldflags -H=windowsgui -o "$OUTPUT_DIR/postwoman-proxy.exe"

  mkdir $OUTPUT_DIR/icons
  cp icons/icon.png "$OUTPUT_DIR/icons/icon.png"

  mkdir $OUTPUT_DIR/data

  rm $OUTPUT_DIR/postwoman-proxy.manifest
  rm rsrc.syso
elif [ "$PLATFORM" == "linux" ]; then
  echo "NOTICE: postwoman-proxy is untested and currently unsupported on Linux."
  GOOS="linux" go build -o "$OUTPUT_DIR/postwoman-proxy"
fi
