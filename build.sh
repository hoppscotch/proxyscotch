#!/bin/sh

TIME_START=$(date +%s)

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


# Load version information
. ./version.properties
echo "Building Proxywoman $BUILD_TYPE v$VERSION_NAME (build $VERSION_CODE) for $PLATFORM"
echo "Developed by @NBTX (Apollo Software)"
echo ""
echo ""


# Ensure output directory exists.
[ -d "out/" ] || mkdir "out/"
[ -d "out/$PLATFORM-$BUILD_TYPE" ] && rm -r "out/$PLATFORM-$BUILD_TYPE"
mkdir "out/$PLATFORM-$BUILD_TYPE"
OUTPUT_DIR="out/$PLATFORM-$BUILD_TYPE"


# Handle special build: server
if [ "$BUILD_TYPE" = "server" ]; then
  echo "Executing go build..."

	if [ "$PLATFORM" = "windows" ]; then
		GOOS="$PLATFORM" go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/proxywoman-server.exe" server/server.go
	  mv "$OUTPUT_DIR/proxywoman-server.exe" "$OUTPUT_DIR/Proxywoman-Server-Windows-v${VERSION_NAME}.exe"
	else
		GOOS="$PLATFORM" go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/proxywoman-server" server/server.go
	  mv "$OUTPUT_DIR/proxywoman-server" "$OUTPUT_DIR/proxywoman-server-${PLATFORM}-v${VERSION_NAME}"
	fi
	exit
fi


# Remove all legacy icons.
[ -f icons/icon_unix.go ] && rm icons/icon_unix.go
[ -f icons/icon_win.go ] && rm icons/icon_win.go


# Build the icon for the appropriate platform.
echo "Generating platform icon..."
if [ "$PLATFORM" = "darwin" ] || [ "$PLATFORM" = "linux" ]; then
  cat "icons/icon.png" | go run github.com/cratonica/2goarray Data icon >> icons/icon_unix.go
elif [ "$PLATFORM" = "windows" ]; then
  cat "icons/icon.ico" | go run github.com/cratonica/2goarray Data icon >> icons/icon_win.go
else
  echo "Unknown platform: $1"
  exit 3
fi


# Copy binary assets.
echo "Copying binary assets..."
cp -r "resources/$PLATFORM/." "$OUTPUT_DIR/"


# Inject placeholders into assets.
echo "Injecting placeholders into binary assets..."
find "$OUTPUT_DIR" -type f -print0 | xargs -0 perl -pi -e "s/\\\$VERSION_NAME/$VERSION_NAME/g"
find "$OUTPUT_DIR" -type f -print0 | xargs -0 perl -pi -e "s/\\\$VERSION_CODE/$VERSION_CODE/g"


# Execute platform build.
echo "Executing go build..."

if [ "$PLATFORM" = "darwin" ]; then
  mkdir -p "$OUTPUT_DIR/Proxywoman.app/Contents/MacOS"
  mkdir -p "$OUTPUT_DIR/Proxywoman.app/Contents/MacOS/icons"
  cp icons/icon.png "$OUTPUT_DIR/Proxywoman.app/Contents/MacOS/icons/"
  GOOS="darwin" GO111MODULE=on go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/Proxywoman.app/Contents/MacOS/postwoman-proxy"

  mv "$OUTPUT_DIR/Proxywoman.app" "$OUTPUT_DIR/Proxywoman-Darwin-v${VERSION_NAME}.app"
elif [ "$PLATFORM" = "windows" ]; then
  [ -f "rsrc.syso" ] && rm rsrc.syso
  go get github.com/akavel/rsrc

  rsrc -manifest="$OUTPUT_DIR/postwoman-proxy.manifest" -ico="icons/icon.ico" -o rsrc.syso
  GOOS="windows" GO111MODULE=on go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE -H=windowsgui" -o "$OUTPUT_DIR/proxywoman.exe"

  mkdir "$OUTPUT_DIR/icons"
  cp icons/icon.png "$OUTPUT_DIR/icons/icon.png"

  mkdir "$OUTPUT_DIR/data"

  rm "$OUTPUT_DIR/postwoman-proxy.manifest"
  rm rsrc.syso

  mv "$OUTPUT_DIR/proxywoman.exe" "$OUTPUT_DIR/Proxywoman-Windows-v${VERSION_NAME}.exe"
elif [ "$PLATFORM" = "linux" ]; then
  echo "NOTICE: Proxywoman is untested and currently unsupported on Linux."
  GOOS="linux" GO111MODULE=on go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/proxywoman"
fi

TIME_END=$(date +%s)
TIME_TAKEN=$(( TIME_END - TIME_START ))

echo ""
echo ""
echo "Done (${TIME_TAKEN}s)."