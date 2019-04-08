package pubsubmodel

import "github.com/ortuman/jackal/xmpp"

type Item struct {
	Publisher string
	Payload   xmpp.Element
}
