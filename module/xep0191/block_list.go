/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0191

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const blockingCommandNamespace = "urn:xmpp:blocking"

const (
	xep191RequestedContextKey = "xep_191:requested"
)

// BlockingCommand returns a blocking command IQ handler module.
type BlockingCommand struct {
	stm stream.C2S
}

// New returns a blocking command IQ handler module.
func New(stm stream.C2S) *BlockingCommand {
	return &BlockingCommand{stm: stm}
}

// RegisterDisco registers disco entity features/items
// associated to blocking command module.
func (x *BlockingCommand) RegisterDisco(discoInfo *xep0030.DiscoInfo) {
	discoInfo.Entity(x.stm.Domain(), "").AddFeature(blockingCommandNamespace)
	discoInfo.Entity(x.stm.JID().ToBareJID().String(), "").AddFeature(blockingCommandNamespace)
}

// MatchesIQ returns whether or not an IQ should be
// processed by the blocking command module.
func (x *BlockingCommand) MatchesIQ(iq *xml.IQ) bool {
	e := iq.Elements()
	blockList := e.ChildNamespace("blocklist", blockingCommandNamespace)
	block := e.ChildNamespace("block", blockingCommandNamespace)
	unblock := e.ChildNamespace("unblock", blockingCommandNamespace)
	return (iq.IsGet() && blockList != nil) || (iq.IsSet() && (block != nil || unblock != nil))
}

// ProcessIQ processes a blocking command IQ taking according actions
// over the associated stream.
func (x *BlockingCommand) ProcessIQ(iq *xml.IQ) {
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

func (x *BlockingCommand) sendBlockList(iq *xml.IQ) {
	blItms, err := storage.Instance().FetchBlockListItems(x.stm.Username())
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	blockList := xml.NewElementNamespace("blocklist", blockingCommandNamespace)
	for _, blItm := range blItms {
		itElem := xml.NewElementName("item")
		itElem.SetAttribute("jid", blItm.JID)
		blockList.AppendElement(itElem)
	}
	reply := iq.ResultIQ()
	reply.AppendElement(blockList)
	x.stm.SendElement(reply)

	x.stm.Context().SetBool(true, xep191RequestedContextKey)
}

func (x *BlockingCommand) block(iq *xml.IQ, block xml.XElement) {
	var bl []model.BlockListItem

	items := block.Elements().Children("item")
	if len(items) == 0 {
		x.stm.SendElement(iq.BadRequestError())
		return
	}
	jds, err := x.extractItemJIDs(items)
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.JidMalformedError())
		return
	}
	blItems, ris, err := x.fetchBlockListAndRosterItems()
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	for _, j := range jds {
		if !x.isJIDInBlockList(j, blItems) {
			x.broadcastPresenceMatchingJID(j, ris, xml.UnavailableType)
			bl = append(bl, model.BlockListItem{Username: x.stm.Username(), JID: j.String()})
		}
	}
	if err := storage.Instance().InsertBlockListItems(bl); err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	router.Instance().ReloadBlockList(x.stm.Username())

	x.stm.SendElement(iq.ResultIQ())
	x.pushIQ(block)
}

func (x *BlockingCommand) unblock(iq *xml.IQ, unblock xml.XElement) {
	items := unblock.Elements().Children("item")
	jds, err := x.extractItemJIDs(items)
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.JidMalformedError())
		return
	}
	blItems, ris, err := x.fetchBlockListAndRosterItems()
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}

	var bl []model.BlockListItem
	if len(jds) == 0 {
		for _, blItem := range blItems {
			j, _ := xml.NewJIDString(blItem.JID, true)
			x.broadcastPresenceMatchingJID(j, ris, xml.AvailableType)
		}
		bl = blItems

	} else {
		for _, j := range jds {
			if x.isJIDInBlockList(j, blItems) {
				x.broadcastPresenceMatchingJID(j, ris, xml.AvailableType)
				bl = append(bl, model.BlockListItem{Username: x.stm.Username(), JID: j.String()})
			}
		}
	}
	if err := storage.Instance().DeleteBlockListItems(bl); err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	router.Instance().ReloadBlockList(x.stm.Username())

	x.stm.SendElement(iq.ResultIQ())
	x.pushIQ(unblock)
}

func (x *BlockingCommand) pushIQ(elem xml.XElement) {
	stms := router.Instance().StreamsMatchingJID(x.stm.JID().ToBareJID())
	for _, stm := range stms {
		if !stm.Context().Bool(xep191RequestedContextKey) {
			continue
		}
		iq := xml.NewIQType(uuid.New(), xml.SetType)
		iq.AppendElement(elem)
		stm.SendElement(iq)
	}
}

func (x *BlockingCommand) broadcastPresenceMatchingJID(jid *xml.JID, ris []model.RosterItem, presenceType string) {
	stms := router.Instance().StreamsMatchingJID(jid)
	for _, stm := range stms {
		if !x.isSubscribedFrom(stm.JID().ToBareJID(), ris) {
			continue
		}
		p := xml.NewPresence(stm.JID(), x.stm.JID().ToBareJID(), presenceType)
		if presence := stm.Presence(); presence != nil && presenceType == xml.AvailableType {
			p.AppendElements(presence.Elements().All())
		}
		router.Instance().MustRoute(p)
	}
}

func (x *BlockingCommand) isJIDInBlockList(jid *xml.JID, blItems []model.BlockListItem) bool {
	for _, blItem := range blItems {
		if blItem.JID == jid.String() {
			return true
		}
	}
	return false
}

func (x *BlockingCommand) isSubscribedFrom(jid *xml.JID, ris []model.RosterItem) bool {
	str := jid.String()
	for _, ri := range ris {
		if ri.JID == str && (ri.Subscription == roster.SubscriptionFrom || ri.Subscription == roster.SubscriptionBoth) {
			return true
		}
	}
	return false
}

func (x *BlockingCommand) fetchBlockListAndRosterItems() ([]model.BlockListItem, []model.RosterItem, error) {
	blItms, err := storage.Instance().FetchBlockListItems(x.stm.Username())
	if err != nil {
		return nil, nil, err
	}
	ris, _, err := storage.Instance().FetchRosterItems(x.stm.Username())
	if err != nil {
		return nil, nil, err
	}
	return blItms, ris, nil
}

func (x *BlockingCommand) extractItemJIDs(items []xml.XElement) ([]*xml.JID, error) {
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
