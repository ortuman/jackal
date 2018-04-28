/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const blockingCommandNamespace = "urn:xmpp:blocking"

const (
	xep191RequestedContextKey = "xep_191:requested"
)

// XEPBlockingCommand returns a blocking command IQ handler module.
type XEPBlockingCommand struct {
	stm           c2s.Stream
	loadedInMemBl bool
	inMemBl       []*xml.JID
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
	e := iq.Elements()
	blockList := e.ChildNamespace("blocklist", blockingCommandNamespace)
	block := e.ChildNamespace("block", blockingCommandNamespace)
	unblock := e.ChildNamespace("unblock", blockingCommandNamespace)
	return (iq.IsGet() && blockList != nil) || (iq.IsSet() && (block != nil || unblock != nil))
}

// ProcessIQ processes a blocking command IQ taking according actions
// over the associated stream.
func (x *XEPBlockingCommand) ProcessIQ(iq *xml.IQ) {
	if err := x.loadInMemBlockList(); err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	if iq.IsGet() {
		x.sendBlockList(iq)
	} else if iq.IsSet() {
		e := iq.Elements()
		if block := e.ChildNamespace("block", blockingCommandNamespace); block != nil {
			x.block(iq, block)
		} else if unblock := e.ChildNamespace("unblock", blockingCommandNamespace); unblock != nil {
			x.unblock(iq, unblock)
		}
	}
}

// IsBlockedJID returns whether or not the passed jid matches any
// of the blocking list JIDs.
func (x *XEPBlockingCommand) IsBlockedJID(jid *xml.JID) bool {
	for _, blockedJID := range x.inMemBl {
		if x.jidMatchesBlockedJID(jid, blockedJID) {
			return true
		}
	}
	return false
}

func (x *XEPBlockingCommand) sendBlockList(iq *xml.IQ) {
	bl := xml.NewElementNamespace("blocklist", blockingCommandNamespace)
	for _, j := range x.inMemBl {
		itElem := xml.NewElementName("item")
		itElem.SetAttribute("jid", j.String())
		bl.AppendElement(itElem)
	}
	reply := iq.ResultIQ()
	reply.AppendElement(bl)
	x.stm.SendElement(reply)

	x.stm.Context().SetBool(true, xep191RequestedContextKey)
}

func (x *XEPBlockingCommand) block(iq *xml.IQ, block xml.XElement) {
	items := block.Elements().Children("item")
	if len(items) == 0 {
		x.stm.SendElement(iq.BadRequestError())
		return
	}
	jids, err := x.extractItemJIDs(items)
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.JidMalformedError())
		return
	}
	var bl []model.BlockListItem
	for _, j := range jids {
		if !x.insertInMemBlockListJID(j) {
			continue
		}
		bl = append(bl, model.BlockListItem{Username: x.stm.Username(), JID: j.String()})
	}
	if len(bl) > 0 {
		if err := storage.Instance().InsertOrUpdateBlockListItems(bl); err != nil {
			log.Error(err)
			x.stm.SendElement(iq.InternalServerError())
			return
		}
	}
	x.stm.SendElement(iq.ResultIQ())
	x.pushIQ(block)
}

func (x *XEPBlockingCommand) unblock(iq *xml.IQ, unblock xml.XElement) {
	items := unblock.Elements().Children("item")
	if len(items) == 0 {
		if err := storage.Instance().DeleteBlockList(x.stm.Username()); err != nil {
			log.Error(err)
			x.stm.SendElement(iq.InternalServerError())
			return
		}
		x.inMemBl = nil

	} else {
		jids, err := x.extractItemJIDs(items)
		if err != nil {
			log.Error(err)
			x.stm.SendElement(iq.JidMalformedError())
			return
		}
		var bl []model.BlockListItem
		for _, j := range jids {
			if !x.deleteInMemBlockListJID(j) {
				continue
			}
			bl = append(bl, model.BlockListItem{Username: x.stm.Username(), JID: j.String()})
		}
		if err := storage.Instance().DeleteBlockListItems(bl); err != nil {
			log.Error(err)
			x.stm.SendElement(iq.InternalServerError())
			return
		}
	}
	x.stm.SendElement(iq.ResultIQ())
	x.pushIQ(unblock)
}

func (x *XEPBlockingCommand) pushIQ(elem xml.XElement) {
	stms := c2s.Instance().AvailableStreams(x.stm.Username())
	for _, stm := range stms {
		if !stm.Context().Bool(xep191RequestedContextKey) {
			continue
		}
		iq := xml.NewIQType(uuid.New(), xml.SetType)
		iq.AppendElement(elem)
		stm.SendElement(iq)
	}
}

func (x *XEPBlockingCommand) loadInMemBlockList() error {
	if x.loadedInMemBl {
		return nil
	}
	bl, err := storage.Instance().FetchBlockListItems(x.stm.Username())
	if err != nil {
		return err
	}
	var blockedJIDs []*xml.JID
	for _, bli := range bl {
		j, _ := xml.NewJIDString(bli.JID, true)
		blockedJIDs = append(blockedJIDs, j)
	}
	x.inMemBl = blockedJIDs
	x.loadedInMemBl = true
	return nil
}

func (x *XEPBlockingCommand) insertInMemBlockListJID(jid *xml.JID) bool {
	for _, j := range x.inMemBl {
		if j.String() == jid.String() {
			return false
		}
	}
	x.inMemBl = append(x.inMemBl, jid)
	return true
}

func (x *XEPBlockingCommand) deleteInMemBlockListJID(jid *xml.JID) bool {
	for i, j := range x.inMemBl {
		if j.String() == jid.String() {
			x.inMemBl = append(x.inMemBl[:i], x.inMemBl[i+1:]...)
			return true
		}
	}
	return false
}

func (x *XEPBlockingCommand) extractItemJIDs(items []xml.XElement) ([]*xml.JID, error) {
	var ret []*xml.JID
	for _, item := range items {
		j, err := xml.NewJIDString(item.Attributes().Get("jid"), false)
		if err != nil {
			return nil, err
		}
		ret = append(ret, j)
	}
	return ret, nil
}

func (x *XEPBlockingCommand) jidMatchesBlockedJID(jid, blockedJID *xml.JID) bool {
	if blockedJID.IsFullWithUser() {
		return jid.Matches(blockedJID, xml.JIDMatchesNode|xml.JIDMatchesDomain|xml.JIDMatchesResource)
	} else if blockedJID.IsBare() {
		return jid.Matches(blockedJID, xml.JIDMatchesNode|xml.JIDMatchesDomain)
	} else if blockedJID.IsServer() && blockedJID.IsFull() {
		return jid.Matches(blockedJID, xml.JIDMatchesDomain|xml.JIDMatchesResource)
	}
	return jid.Matches(blockedJID, xml.JIDMatchesDomain)
}
