# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [0.4.11] - 2019-04-07
### Fixed
- Fixed pgsql private storage

## [0.4.10] - 2019-03-17
### Fixed
- Fallback to standard port on SRV resolution error.
- Use serialization intermediate buffer on socket send.

## [0.4.9] - 2019-02-07
### Fixed
- In-band registration bug.

## [0.4.8] - 2019-01-23
### Fixed
- S2S iq module routing.

## [0.4.7] - 2019-01-22
### Added
- SCRAM-SHA-512 authentication method.

## [0.4.6] - 2019-01-19
### Fixed
- Fixed Gajim client connecting issue.

## [0.4.5] - 2019-01-16
### Added
- PostgreSQL support.

## [0.4.0] - 2019-01-01
### Added
- Cluster mode support. 🥳

## [0.3.6] - 2018-12-15
### Fixed
- Fixed bug in roster item deletion.

## [0.3.5] - 2018-11-09
### Fixed
- Fixed c2s and s2s message routing.

## [0.3.4] - 2018-11-03
### Added
- Built-in graceful shutdown support.

## [0.3.3] - 2018-10-03
### Changed
- New component interface.

## [0.3.2] - 2018-09-04
### Fixed
- Bug fixes.

### Changed
- New module interface.

## [0.3.1] - 2018-07-17
### Fixed
- IQ routing bug.

## [0.3.0] - 2018-07-06
### Added
- Added S2S support.

### Removed
- Removed CGO dependency... thanks Sam Whited! 😉

### Fixed
- crash: invalid XML parsing.

## [0.2.0] - 2018-05-08
### Added
- Added support for XEP-0191 (Blocking Command)
- Added support for XEP-0012 (Last Activity)
- Added support for XEP-0237 (Roster Versioning)
- RFC 7395: XMPP Subprotocol for WebSocket

## [0.1.15] - 2018-03-20
### Added
- Initial release (https://xmpp.org/rfcs/rfc3921.html)
- Added support for XEP-0030 (Service Discovery)
- Added support for XEP-0049 (Private XML Storage)
- Added support for XEP-0054 (vcard-temp)
- Added support for XEP-0077 (In-Band Registration)
- Added support for XEP-0092 (Software Version)
- Added support for XEP-0138 (Stream Compression)
- Added support for XEP-0160 (Best Practices for Handling Offline Messages)
- Added support for XEP-0199 (XMPP Ping)
