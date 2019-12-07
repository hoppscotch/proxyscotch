#!/bin/bash

if [ $# -lt 1 ]; then
  echo "Usage: $0 <unix|win32>"
  exit 3
fi

# Collect parameters.
PLATFORM="$1"

# Remove all legacy icons.
[ -f icons/icon_unix.go ] && rm icons/icon_unix.go
[ -f icons/icon_win.go ] && rm icons/icon_win.go

# Build the icon for the appropriate platform.
if [ "$PLATFORM" == "unix" ]; then
  cat "icons/icon.png" | go run github.com/cratonica/2goarray Data icon >> icons/icon_unix.go
elif [ "$PLATFORM" == "win32" ]; then
  cat "icons/icon.ico" | go run github.com/cratonica/2goarray Data icon >> icons/icon_win.go
else
  echo "Unknown platform: $1"
  exit 3
fi

go run main.go

