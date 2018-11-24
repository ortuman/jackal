/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type messageType int

const (
	messageBindType messageType = iota
	messageUnbindType
	messageSendType
)

type clusterMessage struct {
	typ  messageType
	node string
	jid  *jid.JID
	elem xmpp.XElement
}

func (cm *clusterMessage) fromGob(dec *gob.Decoder) {
	var node, domain, resource string
	dec.Decode(&cm.typ)
	dec.Decode(&cm.node)

	switch cm.typ {
	case messageBindType, messageUnbindType:
		dec.Decode(&node)
		dec.Decode(&domain)
		dec.Decode(&resource)
		cm.jid, _ = jid.New(node, domain, resource, true)

	case messageSendType:
		elem := &xmpp.Element{}
		elem.FromGob(dec)
		cm.elem = elem
	}
}

func (cm *clusterMessage) toGob(enc *gob.Encoder) {
	enc.Encode(cm.typ)
	enc.Encode(cm.node)
	switch cm.typ {
	case messageBindType, messageUnbindType:
		enc.Encode(cm.jid.Node())
		enc.Encode(cm.jid.Domain())
		enc.Encode(cm.jid.Resource())
	case messageSendType:
		cm.elem.ToGob(enc)
	}
}
