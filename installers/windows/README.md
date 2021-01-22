# Windows Installer

The Windows installer includes an [Inno Setup](http://jrsoftware.org/isdl.php) script, that can be used to build an installer. It automatically gathers the necessary files (as specified within the setup script) from the `out/windows-desktop` directory, and allows the user to select the path they would like to use for installation.

> **NOTE**: This Inno Setup script can only be executed using the Inno Setup application, thus this installer can only be built using Windows.

