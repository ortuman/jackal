# jackal

An XMPP server written in Go.

[![Build Status](https://img.shields.io/endpoint.svg?url=https%3A%2F%2Factions-badge.atrox.dev%2Fortuman%2Fjackal%2Fbadge&style=flat)](https://actions-badge.atrox.dev/ortuman/jackal/goto)
[![Go Report Card](https://goreportcard.com/badge/github.com/ortuman/jackal?style=flat-square)](https://goreportcard.com/report/github.com/ortuman/jackal)
[![Godoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/ortuman/jackal)
[![Releases](https://img.shields.io/github/release/ortuman/jackal/all.svg?style=flat-square)](https://github.com/ortuman/jackal/releases)
[![LICENSE](https://img.shields.io/github/license/ortuman/jackal.svg?style=flat-square)](https://github.com/ortuman/jackal/blob/master/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/ortuman/jackal.svg)](https://hub.docker.com/r/ortuman/jackal/)
[![Join the chat at https://gitter.im/jackal-im/jackal](https://badges.gitter.im/jackal-im/jackal.svg)](https://gitter.im/jackal-im/jackal?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

<div align="center">
    <a href="#">
        <img src="./logos/gopher.png">
    </a>
</div>

## About

jackal is a free, open-source, high performance XMPP server which aims to be known for its stability, simple configuration and low resource consumption.

## Features

jackal supports the following features:

- Customizable
- Enforced SSL/TLS
- Stream compression (zlib)
- Database connectivity for storing offline messages and user settings (PostgreSQL 9.5+)
- Clustering capabilities (ectd 3.4+)
- Expose [prometheus](https://prometheus.io/) metrics
- Cross-platform (OS X, Linux)

## Installing

### Getting Started

To start using jackal, install Go 1.16+ and run the following commands:

```bash
$ go get -d github.com/ortuman/jackal
$ cd $GOPATH/src/github.com/ortuman/jackal
$ make install installctl
```

This will fetch the code and install `jackal` and `jackalctl` binaries into your `$GOPATH/bin` path.

By default the application will try to locate service configuration at `config.yaml`, but alternatively you can specify a custom configuration path either through command line.

```sh
$ jackal --config=/your-custom-path/your-config.yaml
```

or environment variable:

```sh
$ env JACKAL_CONFIG_FILE=/your-custom-path/your-config.yaml jackal
```

### PostgreSQL database creation

Create a user and a database for that user:

```sql
CREATE ROLE jackal WITH LOGIN PASSWORD 'password';
CREATE DATABASE jackal;
GRANT ALL PRIVILEGES ON DATABASE jackal TO jackal;
```

Download lastest version of the [PostgreSQL schema](sql/postgres.up.psql) from jackal Github repository.

```sh
wget https://raw.githubusercontent.com/ortuman/jackal/master/sql/postgres.up.psql
```

Run the postgres script file to create database schema:

```sh
psql --user jackal --password -f sql/postgres.up.psql
```

Configure jackal to use PostgreSQL by editing the configuration file:

```yaml
storage:
  type: pgsql
  pgsql:
    host: 127.0.0.1:5432
    user: jackal
    password: password
    database: jackal
```

That's it!

Your database is now ready to connect with jackal.

### Creating jackal user

After completing database setup and starting `jackal` service you'll have to register a new user to be able to login. To do so, you can use
jackal command-line tool to create a new user proving name and password.

```sh
make installctl && jackalctl user add <user>:<password>
```

## Clustering

The purpose of clustering is to be able to use several servers for fault-tolerance and scalability.

Since `jackal` is a distributed system, it needs a distributed data store like [etcd](https://etcd.io/) to share its state across the entire cluster.

To properly run `jackal` in clustering mode make sure to add a `cluster` section configuration in each of your service nodes.

Here's an example of how this section should look like:

```yaml
cluster:
  etcd:
    endpoints:
      - http://<etcd-host1>:<etcd-port1>
      - http://<etcd-host2>:<etcd-port2>
      ...
  port: your-cluster-node-port # default is 14369
```

Note the defined `port` value will be used to perform cluster node communication, so make sure is reachable within your internal network.

## Server extensibility

The purpose of the extensibility framework is to provide an interface between jackal server and third-party external modules, thus offering the possibility of extending the functionality of the service for particular use cases.
Extensibility [gRPC](https://grpc.io/) API proto files can be found at jackal [proto definitions repository](https://github.com/jackal-xmpp/jackal-proto).

* [Authenticators](https://github.com/jackal-xmpp/jackal-proto/blob/master/jackal/proto/authenticator/v1/authenticator.proto#L24-L27)
* [Components](https://xmpp.org/extensions/xep-0114.html)

## Run jackal in Docker

The Docker deployment framework supports easy installation and configuration of jackal server.

You need to have Docker installed on your system before you can use a jackal Docker image. See [Install Docker](https://docs.docker.com/engine/install/) for instructions.

Download the jackal Docker image from the official Docker Hub library with this command:

```sh
docker pull ortuman/jackal:latest
```

Start a new jackal Docker container with custom configuration.

```sh
docker run --name=jackal \
   --mount type=bind,src=/path-on-host-machine/my-custom-config.yaml,dst=/jackal/config.yaml \
   -d ortuman/jackal:latest
```

### Docker compose

Alternatively, and with the purpose of facilitating service mounting, you can make use of `docker-compose` as follows.

```sh
docker-compose -f dockerfiles/docker-compose.yml up
```

This command will spin up a `jackal` server along with its dependencies on its own docker network and start listening for incoming connections on port `5222`.

Once up and running, don't forget to [register one or more users](#creating-jackal-user) using `jackalctl`.

## Supported Specifications
- [RFC 6120: XMPP CORE](https://xmpp.org/rfcs/rfc6120.html)
- [RFC 6121: XMPP IM](https://xmpp.org/rfcs/rfc6121.html)
- [XEP-0004: Data Forms](https://xmpp.org/extensions/xep-0004.html) *2.9*
- [XEP-0012: Last Activity](https://xmpp.org/extensions/xep-0012.html) *2.0*  
- [XEP-0030: Service Discovery](https://xmpp.org/extensions/xep-0030.html) *2.5rc3*
- [XEP-0049: Private XML Storage](https://xmpp.org/extensions/xep-0049.html) *1.2*
- [XEP-0054: vcard-temp](https://xmpp.org/extensions/xep-0054.html) *1.2*
- [XEP-0092: Software Version](https://xmpp.org/extensions/xep-0092.html) *1.1*
- [XEP-0114: Jabber Component Protocol](https://xmpp.org/extensions/xep-0114.html) *1.6*  
- [XEP-0115: Entity Capabilities](https://xmpp.org/extensions/xep-0115.html) *1.5.2*  
- [XEP-0138: Stream Compression](https://xmpp.org/extensions/xep-0138.html) *2.0*
- [XEP-0160: Best Practices for Handling Offline Messages](https://xmpp.org/extensions/xep-0160.html) *1.0.1*
- [XEP-0190: Best Practice for Closing Idle Streams](https://xmpp.org/extensions/xep-0190.html) *1.1*
- [XEP-0191: Blocking Command](https://xmpp.org/extensions/xep-0191.html) *1.3*
- [XEP-0199: XMPP Ping](https://xmpp.org/extensions/xep-0199.html) *2.0*
- [XEP-0202: Entity Time](https://xmpp.org/extensions/xep-0202.html) *2.0*  
- [XEP-0220: Server Dialback](https://xmpp.org/extensions/xep-0220.html) *1.1.1*
- [XEP-0237: Roster Versioning](https://xmpp.org/extensions/xep-0237.html) *1.3*
- [XEP-0280: Message Carbons](https://xmpp.org/extensions/xep-0280.html) *0.13.3*
- [XEP-0368: SRV records for XMPP over TLS](https://xmpp.org/extensions/xep-0368.html) *1.1.0*

## Join and Contribute

The [jackal developer community](https://gitter.im/jackal-im/jackal?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=readme.md) is vital to improving jackal future releases.

Contributions of all kinds are welcome: reporting issues, updating documentation, fixing bugs, improving unit tests, sharing ideas, and any other tips that may help the jackal community.

## Code of Conduct

Help us keep jackal open and inclusive. Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Licensing

jackal is licensed under the Apache 2 License. See
[LICENSE](https://github.com/ortuman/jackal/blob/master/LICENSE) for the full
license text.

## Contact

If you have any suggestion or question:

Miguel Ángel Ortuño, JID: ortuman@jackal.im, email: <ortuman@pm.me>
