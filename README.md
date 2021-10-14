<div align="center">
  <a href="https://hoppscotch.io"><img src="https://hoppscotch.io/icon.png" alt="Hoppscotch" height="128"></a>
  <br>
  <h1>Proxyscotch</h1>
  <p>
    API request builder - Helps you create your requests faster, saving you precious time on your development.
  </p>
</div>

---

A simple proxy server created by [@SamJakob](https://github.com/SamJakob/) for [Hoppscotch](https://github.com/hoppscotch/hoppscotch/) and formerly hosted by [Apollo Software Limited](https://apollosoftware.xyz/).

## Installation 📦
**Proxyscotch requires `zenity` on Linux. This is available in most distribution package managers.**

We're still working on automated installers. For now, copy the binary to a user-writeable location and launch the application.  
A dialog will open and explain the certificate installation process - there are more detailed instructions in our [wiki](https://github.com/hoppscotch/proxyscotch/wiki).

## Demo 🚀
[https://hoppscotch.io](https://hoppscotch.io)


## Building 🏗️

*These are bash scripts. In order to execute them on Windows, you will need to use some form of bash shell on Windows. We recommend [Git Bash](https://gitforwindows.org/).*

- macOS:
```bash
# To build the desktop tray application:
$ ./build.sh darwin

# To build the server application:
$ ./build.sh darwin server
```

- For Linux desktops:
```bash
# To build the desktop tray application:
$ ./build.sh linux

# To build the server application:
$ ./build.sh linux server
```

- For Windows desktops:
```bash
# To build the desktop tray application:
$ ./build.sh windows

# To build the server application:
$ ./build.sh windows server
```

> The build output is placed in the `out/` directory.



## Installers 🧙
The `installers/` directory contains scripts for each platform to generate an installer application.  
Each platform's installer directory, contains the relevant information for that installer.
- [macOS](installers/darwin)
- [Windows](installers/windows)
- [Linux](installers/linux)



## Usage 👨‍💻
### Desktops 🖥️
The proxy will add a tray icon to the native system tray for your platform, which will contain all of the options for the proxy.

### Servers 🖧
To use the proxy on a server, clone the package, build the server using the instructions above, and use:
```bash
$ ./out/<platform>-server/server --host="<hostname>:<port>" --token="<token_or_blank>"

# e.g. on Linux
$ ./out/linux-server/server --host="<hostname>:<port>" --token="<token_or_blank>"

# or on Windows
$ ./out/windows-server/server.exe --host="<hostname>:<port>" --token="<token_or_blank>"
```

- The `host` and `token` parameters are optional. The defaults are as follows:
- `host`: `localhost:9159`
- `token`: blank; allowing anyone to access (see below)

**NOTE:** When the token is blank it will allow *anybody* to access your proxy server. This may be what you want, but please be sure to consider the security implications.
