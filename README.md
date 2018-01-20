# jackal
A XMPP server written in Go.

[![License](https://img.shields.io/badge/license-GPL-blue.svg)](https://github.com/ortuman/jackal/blob/master/LICENSE)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/ortuman/jackal/issues)

<div align="center">
    <a href="#">
        <img src="./doc/gopher.png">
    </a>
</div>

## About
Jackal is a modern XMPP communication server. It aims to be easy to set up and configure, and efficient with system resources. Additionally, for developers it aims to be easy to extend and give a flexible system on which to rapidly develop added functionality, or prototype new protocols.

## Installing

To start using jackal, install Go and run `go get`:

```sh
$ go get github.com/ortuman/jackal
```

This will retrieve the code and install the `jackal` server application into your `$GOBIN` path.

By default the application will try to read server configuration from `/etc/jackal/jackal.yaml` file, but alternatively you can specify a custom configuration path from command line.

```sh
$ jackal --config=$GOPATH/src/github.com/ortuman/jackal/example.jackal.yaml
```

## Features

## XMPP Extension Protocol
- [XEP-0030 Service Discovery](https://xmpp.org/extensions/xep-0030.html)
- [XEP-0049 Private XML Storage](https://xmpp.org/extensions/xep-0049.html)
- [XEP-0054 vcard-temp](https://xmpp.org/extensions/xep-0054.html)
- [XEP-0077 In-Band Registration](https://xmpp.org/extensions/xep-0077.html)
- [XEP-0092 Software Version](https://xmpp.org/extensions/xep-0092.html)
- [XEP-0138 Stream Compression](https://xmpp.org/extensions/xep-0138.html)
- [XEP-0160: Best Practices for Handling Offline Messages](https://xmpp.org/extensions/xep-0160.html)
- [XEP-0199 XMPP Ping](https://xmpp.org/extensions/xep-0199.html)

## Licensing

Jackal is licensed under the GNU General Public License, Version 3.0. See
[LICENSE](https://github.com/ortuman/jackal/blob/master/LICENSE) for the full
license text.
