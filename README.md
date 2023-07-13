
# CmdWebCtrl

A utility that runs a given command and provides a simple web interface for viewing its output to the console and restarting the process. The original use case was easy restarts of the `helics_broker_server` command when errors occurred, without needing to give out SSH access to the server.


## Installation

Install from source using `go install`:

```bash
  go install github.com/gmlc-tdc/cmdwebctrl@latest
```
    

## Usage

Create a `config.toml` file in `$HOME/.cmdwebctrl` or the same directory as the `cmdwebctrl` binary.

The below example will make `cmdwebctrl` accessible using port 8080, and run the command `grep -inr "hello world"` as soon as it starts:

```toml
ServerAddress=":8080"
Command="grep"
Args=["-inr", "hello world"]
RunOnLaunch=true
StdoutToTerminal=false
StderrToTermianl=false
Password="averysecurepassword"
```

The web interface can then be accessed on the same machine at `localhost:8080`.

Some operating systems will restrict binding to low port numbers -- a web server such as Caddy can be set up as a [reverse proxy](https://caddyserver.com/docs/quick-starts/reverse-proxy) to allow accessing `cmdwebctrl` over standard HTTP or HTTPS ports, so `cmdwebctrl` does not need to be run as root. Alternatively on Linux `cmdwebctrl` could be given the `CAP_NET_BIND_SERVICE` capability via `sudo /sbin/setcap 'cap_net_bind_service=ep'`, which will allow it to use low port numbers.


## License

CmdWebCtrl is released under the MIT license. See the [LICENSE](./LICENSE)
and [NOTICE](./NOTICE) files for details. All new contributions must be made
under this license.

SPDX-License-Identifier: MIT

LLNL-CODE-851427
