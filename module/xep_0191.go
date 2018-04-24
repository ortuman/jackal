/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const blockingCommandNamespace = "urn:xmpp:blocking"

const (
	xep191RequestedContextKey = "xep_191:requested"
)

type XEPBlockingCommand struct {
	stm c2s.Stream
}

// NewXEPBlockingCommand returns a blocking command IQ handler module.
func NewXEPBlockingCommand(stm c2s.Stream) *XEPBlockingCommand {
	return &XEPBlockingCommand{stm: stm}
}

// AssociatedNamespaces returns namespaces associated
// with blocking command module.
func (x *XEPBlockingCommand) AssociatedNamespaces() []string {
	return []string{blockingCommandNamespace}
}

// Done signals stream termination.
func (x *XEPBlockingCommand) Done() {
}

// MatchesIQ returns whether or not an IQ should be
// processed by the blocking command module.
func (x *XEPBlockingCommand) MatchesIQ(iq *xml.IQ) bool {
	return iq.Elements().ChildNamespace("blocklist", blockingCommandNamespace) != nil
}

// ProcessIQ processes a blocking command IQ taking according actions
// over the associated stream.
func (x *XEPBlockingCommand) ProcessIQ(iq *xml.IQ) {
	blockList := iq.Elements().ChildNamespace("blocklist", blockingCommandNamespace)
	if iq.IsGet() {
		x.sendBlockList(iq)
	} else if iq.IsSet() {
		x.blockJIDs(iq, blockList)
	}
}

func (x *XEPBlockingCommand) sendBlockList(iq *xml.IQ) {
	items, err := storage.Instance().FetchBlockListItems(x.stm.Username())
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	bl := xml.NewElementNamespace("blocklist", blockingCommandNamespace)
	for _, item := range items {
		itElem := xml.NewElementName("item")
		itElem.SetAttribute("jid", item.JID)
		bl.AppendElement(itElem)
	}
	reply := iq.ResultIQ()
	reply.AppendElement(bl)
	x.stm.SendElement(reply)

	x.stm.Context().SetBool(true, xep191RequestedContextKey)
}

func (x *XEPBlockingCommand) blockJIDs(iq *xml.IQ, blockList xml.XElement) {
}
