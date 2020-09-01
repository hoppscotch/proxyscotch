#!/bin/sh

#
# Basic Prep...
#

TIME_START=$(date +%s)

fcomplete() {
  TIME_END=$(date +%s)
  TIME_TAKEN=$(( TIME_END - TIME_START ))

  echo "Done (${TIME_TAKEN}s)."
  exit 0
}

# Ensure the user is in the correct directory (the directory containing this script.)
if [ "$(pwd)" != "${0%/*}" ]; then
  cd "${0%/*}" || exit
fi

. ./version.properties

#
# Print Banner
#
echo "______                                  _       _     ";
echo "| ___ \                                | |     | |    ";
echo "| |_/ / __ _____  ___   _ ___  ___ ___ | |_ ___| |__  ";
echo "|  __/ '__/ _ \ \/ / | | / __|/ __/ _ \| __/ __| '_ \ ";
echo "| |  | | | (_) >  <| |_| \__ \ (_| (_) | || (__| | | |";
echo "\_|  |_|  \___/_/\_\\__, |___/\___\___/ \__\___|_| |_|";
echo "                     __/ |                            ";
echo "                    |___/                             ";
printf "\n";
printf "v%s -- a Hoppscotch.io project - https://hoppscotch.io/\n" "$VERSION_NAME";
printf "Built by NBTX (Apollo Software) - https://apollosoftware.xyz/\n";
printf "\n";
printf "\n";

#
# Start Build Script
#

if [ $# -lt 1 ]; then
  printf "Usage: %s <darwin|linux|windows> [<server|desktop>]   -  Builds Proxyscotch for the given platform.\n" "$0"
  printf "Usage: %s clean                                       -  Cleans the out/ directory.\n" "$0"
  exit 3
fi

#
# COMMAND: clean
#
if [ "$1" = "clean" ]; then
  echo "Cleaning build directory..."
  rm -rf ./out/*

  fcomplete
  exit 0
fi

#
# COMMAND: publish
#
if [ "$1" = "publish" ]; then
  RELEASE_TAG_NAME="v$VERSION_NAME"

  CONFIRMED=0
  case "$@[@]" in *"-c"*) CONFIRMED=1 ;; esac

  echo "Validating..."
  if GIT_DIR=./.git git rev-parse "$RELEASE_TAG_NAME" >/dev/null 2>&1; then
      echo "Version $VERSION_NAME already exists! (perhaps you need to bump version name and code.)"
      exit 1
  fi

  echo "Validation succeeded."
  echo ""


  if [ "$CONFIRMED" -ne 1 ]; then
    echo "========================================"
    echo "You are preparing the following release:"
    echo "========================================"
    echo ""
    printf "Version Name:\t\t%s\n" "$VERSION_NAME"
    printf "Version Code:\t\t%s\n" "$VERSION_CODE"
    echo ""
    echo "Before pushing, please ensure you have:"
    echo "- tested your build thoroughly on"
    echo "  all supported systems."
    echo "- sufficiently selected and/or bumped"
    echo "  the version number for your release."
    echo ""
    echo ""
    echo "To confirm you have done this, please"
    echo "run the same command again, specifying"
    echo "-c."
    exit 0
  fi

  echo "Preparing new release..."
  git tag -a "$RELEASE_TAG_NAME"

  echo "Pushing release..."
  git push origin "$RELEASE_TAG_NAME"

  exit 0
fi

#
# COMMAND: unpublish
#
if [ "$1" = "unpublish" ]; then
  echo "Fetching all releases..."
  git fetch

  RELEASE_TAG_NAME="v$VERSION_NAME"

  CONFIRMED=0
  case "$@[@]" in *"-c"*) CONFIRMED=1 ;; esac

  echo "Validating..."
  if ! GIT_DIR=./.git git rev-parse "$RELEASE_TAG_NAME" >/dev/null 2>&1; then
      echo "Version $VERSION_NAME doesn't exist."
      exit 1
  fi

  echo "Validation succeeded."
  echo ""


  if [ "$CONFIRMED" -ne 1 ]; then
    echo "=============================================="
    echo "You are about to remove the following release:"
    echo "=============================================="
    echo ""
    printf "Published Tag:\t\t%s\n" "$RELEASE_TAG_NAME"
    printf "Version Name:\t\t%s\n" "$VERSION_NAME"
    printf "Version Code:\t\t%s\n" "$VERSION_CODE"
    echo ""
    echo "To confirm you wish to proceed, please run the"
    echo "same command again, specifying -c."
    exit 0
  fi

  echo "Unpublishing release."
  git tag -d "$RELEASE_TAG_NAME"
  git push origin ":refs/tags/$RELEASE_TAG_NAME"

  exit 0
fi

#
# COMMAND (implicit): build
#

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


# We're running a build.
echo "Building Proxyscotch $BUILD_TYPE v$VERSION_NAME (build $VERSION_CODE) for $PLATFORM"
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
		GOOS="$PLATFORM" go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/proxyscotch-server.exe" server/server.go
	  mv "$OUTPUT_DIR/proxyscotch-server.exe" "$OUTPUT_DIR/proxyscotch-server-windows-v${VERSION_NAME}.exe"

    # echo "Compressing release binary..."
    # WORKING_DIR=$(pwd)
    # cd "$OUTPUT_DIR" || exit 1
	  # zip -r "proxyscotch-server-windows-v${VERSION_NAME}.zip" "proxyscotch-server-windows-v${VERSION_NAME}.exe"
	  # cd "$WORKING_DIR" || exit 1
	else
		GOOS="$PLATFORM" go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/proxyscotch-server" server/server.go
	  mv "$OUTPUT_DIR/proxyscotch-server" "$OUTPUT_DIR/proxyscotch-server-${PLATFORM}-v${VERSION_NAME}"

	  # echo "Compressing release binary..."
	  # WORKING_DIR=$(pwd)
    # cd "$OUTPUT_DIR" || exit 1
	  # zip -r "proxyscotch-server-${PLATFORM}-v${VERSION_NAME}.zip" "proxyscotch-server-${PLATFORM}-v${VERSION_NAME}"
	  # cd "$WORKING_DIR" || exit 1
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
  mkdir -p "$OUTPUT_DIR/Proxyscotch.app/Contents/MacOS"
  mkdir -p "$OUTPUT_DIR/Proxyscotch.app/Contents/MacOS/icons"
  cp icons/icon.png "$OUTPUT_DIR/Proxyscotch.app/Contents/MacOS/icons/"
  GOOS="darwin" GO111MODULE=on go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/Proxyscotch.app/Contents/MacOS/proxyscotch"

  # Produce output binaries
  mv "$OUTPUT_DIR/Proxyscotch.app" "$OUTPUT_DIR/Proxyscotch-macOS-v${VERSION_NAME}.app"

  # Compressing output binaries
  echo "Compressing output binaries"

  WORKING_DIR=$(pwd)
  cd "$OUTPUT_DIR" || exit 1
  zip -r "Proxyscotch-macOS-v${VERSION_NAME}.zip" "Proxyscotch-macOS-v${VERSION_NAME}.app"

  cd "$WORKING_DIR" || exit 1
elif [ "$PLATFORM" = "windows" ]; then
  [ -f "rsrc.syso" ] && rm rsrc.syso
  go get github.com/akavel/rsrc

  rsrc -manifest="$OUTPUT_DIR/proxyscotch.manifest" -ico="icons/icon.ico" -o rsrc.syso
  GOOS="windows" GO111MODULE=on go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE -H=windowsgui" -o "$OUTPUT_DIR/proxyscotch.exe"

  mkdir "$OUTPUT_DIR/icons"
  cp icons/icon.png "$OUTPUT_DIR/icons/icon.png"

  mkdir "$OUTPUT_DIR/data"

  rm "$OUTPUT_DIR/proxyscotch.manifest"
  rm rsrc.syso

  mv "$OUTPUT_DIR/proxyscotch.exe" "$OUTPUT_DIR/Proxyscotch-Windows-v${VERSION_NAME}.exe"

  # Compressing output binaries
  echo "Compressing output binaries"

  # WORKING_DIR=$(pwd)
  # cd "$OUTPUT_DIR" || exit 1
  # zip -r "Proxyscotch-Windows-v${VERSION_NAME}.zip" "Proxyscotch-Windows-v${VERSION_NAME}.exe"
  # cd "$WORKING_DIR" || exit 1
elif [ "$PLATFORM" = "linux" ]; then
  GOOS="linux" GO111MODULE=on go build -ldflags "-X main.VersionName=$VERSION_NAME -X main.VersionCode=$VERSION_CODE" -o "$OUTPUT_DIR/Proxyscotch-Linux-v${VERSION_NAME}"

  # Compressing output binaries
  # echo "Compressing output binaries"
  # WORKING_DIR=$(pwd)
  # cd "$OUTPUT_DIR" || exit 1
  # zip -r "Proxyscotch-Linux-v${VERSION_NAME}.zip" "Proxyscotch-Linux-v${VERSION_NAME}"
  # cd "$WORKING_DIR" || exit 1
fi

echo ""
echo ""
fcomplete