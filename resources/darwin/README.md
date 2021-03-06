# Proxyscotch: Darwin Build

## resources/darwin

This directory acts as a skeleton, used to build an application container for Darwin.  
The format for Darwin applications, is a directory with extension `.app`, containing a manifest and a `Contents/Resources` directory, as well as a `Contents/MacOS` directory containing a `Mach-O` executable.

This resources directory contains the application manifest (`Contents/Info.plist`) and the resources directory (`Contents/Resources`). The Mach-O executable and corresponding directory are then added to the skeleton when it is copied to the build directory.
