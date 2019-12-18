<div align="center">
  <a href="https://postwoman.io"><img src="https://postwoman.io/icons/logo.svg" alt="Postwoman" height="128"></a>
  <br>
  <h1>Postwoman Proxy</h1>
  <p>
    API request builder - Helps you create your requests faster, saving you precious time on your development.
  </p>
</div>

---

A simple proxy server created by [@NBTX](https://github.com/NBTX/) for [Postwoman](https://github.com/liyasthomas/postwoman/) and hosted by [ApolloTV](https://apollotv.xyz/).

## Demo 🚀
[https://postwoman.io](https://postwoman.io)

## Building
*These build scripts are for macOS/Linux systems. Currently, Windows build scripts have not yet been created.*

- For macOS desktops:
```bash
$ ./build.sh darwin
```

- For Linux desktops:
```bash
$ ./build.sh linux
```

- For Windows desktops:
```bash
$ ./build.sh windows
```

- For servers:
```bash
$ go build server/server.go
```

## Usage
To use the proxy on a server, clone the package and use:
```bash
$ ./server --host="<hostname>:<port>" --token="<token_or_blank>"
```

- The `host` and `token` parameters are optional. The defaults are as follows:
- `host`: `localhost:9159`
- `token`: blank; allowing anyone to access (see below)

**NOTE:** When the token is blank it will allow *anybody* to access your proxy server. This may be what you want, but do keep that in mind.
