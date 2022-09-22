# Changelog

## jackal - main / unreleased

## 0.62.1 (2022/09/22)

* [BUGFIX] storage/archive: fix timestamp precision [#254](https://github.com/ortuman/jackal/pull/254)

## 0.62.0 (2022/09/13)

* [FEATURE] module: added support for xep-0313 [#241](https://github.com/ortuman/jackal/pull/241), [#253](https://github.com/ortuman/jackal/pull/253)
* [ENHANCEMENT] auth: re-enable TLS 1.3 channel binding during auth using [RFC 9266](https://www.rfc-editor.org/rfc/rfc9266) [#247](https://github.com/ortuman/jackal/pull/247)
* [ENHANCEMENT] hook: include propagated context into execution parameter. [#249](https://github.com/ortuman/jackal/pull/249)
* [ENHANCEMENT] transport: limit writer buffer size [#251](https://github.com/ortuman/jackal/pull/251)

## 0.61.0 (2022/06/06)

* [ENHANCEMENT] Helm: added support for cloud LB. [237](https://github.com/ortuman/jackal/pull/237) 

## 0.60.0 (2022/05/27)

* [ENHANCEMENT] Helm chart. [#217](https://github.com/ortuman/jackal/pull/217)
* [ENHANCEMENT] Improve k8s compatibility. [#219](https://github.com/ortuman/jackal/pull/219), [#220](https://github.com/ortuman/jackal/pull/220), [#221](https://github.com/ortuman/jackal/pull/221), [#222](https://github.com/ortuman/jackal/pull/222), [#223](https://github.com/ortuman/jackal/pull/223), [#224](https://github.com/ortuman/jackal/pull/224), [#225](https://github.com/ortuman/jackal/pull/225), [#226](https://github.com/ortuman/jackal/pull/226), [#227](https://github.com/ortuman/jackal/pull/227), [#228](https://github.com/ortuman/jackal/pull/228), [#229](https://github.com/ortuman/jackal/pull/229), [#230](https://github.com/ortuman/jackal/pull/230), [#231](https://github.com/ortuman/jackal/pull/231), [#232](https://github.com/ortuman/jackal/pull/232)

## 0.58.0 (2022/03/04)

* [FEATURE] Added BoltDB repository type. [#212](https://github.com/ortuman/jackal/pull/212)

## 0.57.0 (2022/02/12)

* [ENHANCEMENT] Added memory ballast. [#198](https://github.com/ortuman/jackal/pull/198)
* [ENHANCEMENT] Added support for Redis cached repository. [#202](https://github.com/ortuman/jackal/pull/202)
* [ENHANCEMENT] Cached VCard repository. [#203](https://github.com/ortuman/jackal/pull/203)
* [ENHANCEMENT] Cached Last repository. [#204](https://github.com/ortuman/jackal/pull/204)
* [ENHANCEMENT] Cached Capabilities repository. [#205](https://github.com/ortuman/jackal/pull/205)
* [ENHANCEMENT] Cached Private repository. [#206](https://github.com/ortuman/jackal/pull/206)
* [ENHANCEMENT] Cached BlockList repository. [#207](https://github.com/ortuman/jackal/pull/207) 
* [ENHANCEMENT] Cached Roster repository. [#208](https://github.com/ortuman/jackal/pull/208)
* [CHANGE] Introduced measured repository transaction type. [#200](https://github.com/ortuman/jackal/pull/200)
* [CHANGE] Use PgSQL locker. [#201](https://github.com/ortuman/jackal/pull/201)
* [BUGFIX] Fix S2S db key check when nop KV is used. [#199](https://github.com/ortuman/jackal/pull/199)

