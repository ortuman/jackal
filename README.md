# jackal

An XMPP server written in Go.

[![Build Status](https://travis-ci.org/ortuman/jackal.svg?branch=master)](https://travis-ci.org/ortuman/jackal)
[![GoDoc](https://godoc.org/github.com/ortuman/jackal?status.svg)](https://godoc.org/github.com/ortuman/jackal)
[![Test Coverage](https://api.codeclimate.com/v1/badges/e3bcd6e00a2f4493e175/test_coverage)](https://codeclimate.com/github/ortuman/jackal/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/e3bcd6e00a2f4493e175/maintainability)](https://codeclimate.com/github/ortuman/jackal/maintainability)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/8e1575d0e64141a8bd4f8656e44052e6)](https://www.codacy.com/app/ortuman/jackal?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=ortuman/jackal&amp;utm_campaign=Badge_Grade)
[![Go Report Card](https://goreportcard.com/badge/github.com/ortuman/jackal)](https://goreportcard.com/report/github.com/ortuman/jackal)
[![License](https://img.shields.io/badge/license-GPL-blue.svg)](https://github.com/ortuman/jackal/blob/master/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/ortuman/jackal.svg)](https://hub.docker.com/r/ortuman/jackal/)
[![Join the chat at https://gitter.im/jackal-im/jackal](https://badges.gitter.im/jackal-im/jackal.svg)](https://gitter.im/jackal-im/jackal?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

<div align="center">
    <a href="#">
        <img src="./.github/gopher.png">
    </a>
</div>

## About

jackal is a free, open-source, high performance XMPP server which aims to be known for its stability, simple configuration and low resource consumption.

## Features

jackal supports the following features:

- Customizable
- Enforced SSL/TLS
- Stream compression (zlib)
- Database connectivity for storing offline messages and user settings ([BadgerDB](https://github.com/dgraph-io/badger), MySQL 5.7+, MariaDB 10.2+, PostgreSQL 9.5+)
- Cross-platform (OS X, Linux)

## Installing

### Getting Started

To start using jackal, install Go 1.12+ and run the following commands:

```bash
$ go get -d github.com/ortuman/jackal
$ cd $GOPATH/src/github.com/ortuman/jackal
$ make install
```

This will retrieve the code and install the `jackal` server application into your `$GOPATH/bin` path.

By default the application will try to read server configuration from `/etc/jackal/jackal.yml` file, but alternatively you can specify a custom configuration path from command line.

```sh
$ jackal --config=$GOPATH/src/github.com/ortuman/jackal/example.jackal.yml
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

Download lastest version of the [MySQL schema](sql/mysql.up.sql) from jackal Github repository.

```sh
wget https://raw.githubusercontent.com/ortuman/jackal/master/sql/mysql.sql
```

Load database schema into the database.

```sh
mysql -h localhost -D jackal -u jackal -p < mysql.sql
```

Your database is now ready to connect with jackal.

### Using PostgreSQL

Create a user and a database for that user:

```sql
CREATE ROLE jackal WITH LOGIN PASSWORD 'password';
CREATE DATABASE jackal;
GRANT ALL PRIVILEGES ON DATABASE jackal TO jackal;
```

Run the postgres script file to create database schema. In jackal's root directory run:

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

### Cluster configuration

The purpose of clustering is to be able to use several servers for fault-tolerance and scalability.

To run `jackal` in clustering mode make sure to add a `cluster` section configuration in each of the cluster nodes.

Here is an example of how this section should look like:
```yaml
cluster:
  name: node1                             
  port: 5010                              
  hosts: [node2:5010, node3:5010] 
```

Do not forget to include all cluster nodes, excluding the local one, in the `hosts` array field. Otherwise the expected behavior will be undefined.   

## Run jackal in Docker

Set up `jackal` in the cloud in under 5 minutes with zero knowledge of Golang or Linux shell using our [jackal Docker image](https://hub.docker.com/r/ortuman/jackal/).

```bash
$ docker pull ortuman/jackal
$ docker run --name jackal -p 5222:5222 ortuman/jackal
```

## Supported Specifications
- [RFC 6120: XMPP CORE](https://xmpp.org/rfcs/rfc6120.html)
- [RFC 6121: XMPP IM](https://xmpp.org/rfcs/rfc6121.html)
- [RFC 7395: XMPP Subprotocol for WebSocket](https://tools.ietf.org/html/rfc7395)
- [XEP-0004: Data Forms](https://xmpp.org/extensions/xep-0004.html) *2.9*
- [XEP-0012: Last Activity](https://xmpp.org/extensions/xep-0012.html) *2.0*
- [XEP-0030: Service Discovery](https://xmpp.org/extensions/xep-0030.html) *2.5rc3*
- [XEP-0049: Private XML Storage](https://xmpp.org/extensions/xep-0049.html) *1.2*
- [XEP-0054: vcard-temp](https://xmpp.org/extensions/xep-0054.html) *1.2*
- [XEP-0077: In-Band Registration](https://xmpp.org/extensions/xep-0077.html) *2.4*
- [XEP-0092: Software Version](https://xmpp.org/extensions/xep-0092.html) *1.1*
- [XEP-0138: Stream Compression](https://xmpp.org/extensions/xep-0138.html) *2.0*
- [XEP-0160: Best Practices for Handling Offline Messages](https://xmpp.org/extensions/xep-0160.html) *1.0.1*
- [XEP-0191: Blocking Command](https://xmpp.org/extensions/xep-0191.html) *1.3*
- [XEP-0199: XMPP Ping](https://xmpp.org/extensions/xep-0199.html) *2.0*
- [XEP-0220: Server Dialback](https://xmpp.org/extensions/xep-0220.html) *1.1.1*
- [XEP-0237: Roster Versioning](https://xmpp.org/extensions/xep-0237.html) *1.3*

## Join and Contribute

The [jackal developer community](https://gitter.im/jackal-im/jackal?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=readme.md) is vital to improving jackal future releases.  

Contributions of all kinds are welcome: reporting issues, updating documentation, fixing bugs, improving unit tests, sharing ideas, and any other tips that may help the jackal community.

## Code of Conduct

Help us keep jackal open and inclusive. Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Licensing

jackal is licensed under the GNU General Public License, Version 3.0. See
[LICENSE](https://github.com/ortuman/jackal/blob/master/LICENSE) for the full
license text.

## Contact

If you have any suggestion or question:

Miguel Ángel Ortuño, JID: ortuman@jackal.im, email: <ortuman@pm.me>
