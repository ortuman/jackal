# jackal

An XMPP server written in Go.

[![Build Status](https://travis-ci.org/ortuman/jackal.svg?branch=master)](https://travis-ci.org/ortuman/jackal)
[![GoDoc](https://godoc.org/github.com/ortuman/jackal?status.svg)](https://godoc.org/github.com/ortuman/jackal)
[![codecov](https://codecov.io/gh/ortuman/jackal/branch/master/graph/badge.svg)](https://codecov.io/gh/ortuman/jackal)
[![codebeat badge](https://codebeat.co/badges/6573d819-3ef7-4761-9b0b-410264784d8b)](https://codebeat.co/projects/github-com-ortuman-jackal-master)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/8e1575d0e64141a8bd4f8656e44052e6)](https://www.codacy.com/app/ortuman/jackal?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=ortuman/jackal&amp;utm_campaign=Badge_Grade)
[![Go Report Card](https://goreportcard.com/badge/github.com/ortuman/jackal)](https://goreportcard.com/report/github.com/ortuman/jackal)
[![License](https://img.shields.io/badge/license-MPL-blue.svg)](https://github.com/ortuman/jackal/blob/master/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/ortuman/jackal.svg)](https://hub.docker.com/r/ortuman/jackal/)
[![Join the chat at https://gitter.im/jackal-im/jackal](https://badges.gitter.im/jackal-im/jackal.svg)](https://gitter.im/jackal-im/jackal?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

<div align="center">
    <a href="#">
        <img src="./doc/gopher.png">
    </a>
</div>

## About

jackal is a free, open-source, high performance XMPP server which aims to be known for its stability, simple configuration and low resource consumption.

## Features

jackal supports the following features:

- Customizable
- Enforced SSL/TLS
- Stream compression (zlib)
- Database connectivity for storing offline messages and user settings ([BadgerDB](https://github.com/dgraph-io/badger), MySQL 5.7.x)
- Cross-platform (OS X, Linux)

## Installing

### Getting Started

To start using jackal, install Go 1.9+ and run `go get`:

```sh
$ go get github.com/ortuman/jackal
```

This will retrieve the code and install the `jackal` server application into your `$GOBIN` path.

By default the application will try to read server configuration from `/etc/jackal/jackal.yaml` file, but alternatively you can specify a custom configuration path from command line.

```sh
$ jackal --config=$GOPATH/src/github.com/ortuman/jackal/example.jackal.yaml
```

### MySQL database creation

Grant right to a dedicated 'jackal' user (replace `password` with your desired password).

```sh
echo "GRANT ALL ON jackal.* TO 'jackal'@'localhost' IDENTIFIED BY 'password';" | mysql -h localhost -u root -p
```

Create 'jackal' database (using previously created password).

```sh
echo "CREATE DATABASE jackal;" | mysql -h localhost -u jackal -p
```

Download lastest version of the [MySQL schema](./sql/mysql.sql) from jackal Github repository.

```sh
wget https://raw.githubusercontent.com/ortuman/jackal/master/sql/mysql.sql
```

Load database schema into the database.

```sh
mysql -h localhost -D jackal -u jackal -p < mysql.sql
```

Your database is now ready to connect with jackal.

## Run jackal in Docker

Set up `jackal` in the cloud in under 5 minutes with zero knowledge of Golang or Linux shell using our [jackal Docker image](https://hub.docker.com/r/ortuman/jackal/).

```bash
$ docker pull ortuman/jackal
$ docker run --name jackal -p 5222:5222 ortuman/jackal
```

## XMPP Extension Protocol
- [XEP-0030 Service Discovery](https://xmpp.org/extensions/xep-0030.html)
- [XEP-0049 Private XML Storage](https://xmpp.org/extensions/xep-0049.html)
- [XEP-0054 vcard-temp](https://xmpp.org/extensions/xep-0054.html)
- [XEP-0077 In-Band Registration](https://xmpp.org/extensions/xep-0077.html)
- [XEP-0092 Software Version](https://xmpp.org/extensions/xep-0092.html)
- [XEP-0138 Stream Compression](https://xmpp.org/extensions/xep-0138.html)
- [XEP-0160: Best Practices for Handling Offline Messages](https://xmpp.org/extensions/xep-0160.html)
- [XEP-0199 XMPP Ping](https://xmpp.org/extensions/xep-0199.html)

## Join and Contribute

The [jackal developer community](https://gitter.im/jackal-im/jackal?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=readme.md) is vital to improving jackal future releases.  

Contributions of all kinds are welcome: reporting issues, updating documentation, fixing bugs, improving unit tests, sharing ideas, and any other tips that may help the jackal community.

## Licensing

jackal is licensed under the Mozilla Public License, Version 2.0. See
[LICENSE](https://github.com/ortuman/jackal/blob/master/LICENSE) for the full
license text.

## Contact

If you have any suggestion or question:

Miguel Ángel Ortuño, <ortuman@protonmail.com>
