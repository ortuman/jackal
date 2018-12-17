/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0191

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
)

const mailboxSize = 2048

const blockingCommandNamespace = "urn:xmpp:blocking"

const (
	xep191RequestedContextKey = "xep_191:requested"
)

// BlockingCommand returns a blocking command IQ handler module.
type BlockingCommand struct {
	router     *router.Router
	roster     *roster.Roster
	actorCh    chan func()
	shutdownCh chan chan bool
}

// New returns a blocking command IQ handler module.
func New(disco *xep0030.DiscoInfo, roster *roster.Roster, router *router.Router) (*BlockingCommand, chan<- chan bool) {
	b := &BlockingCommand{
		router:     router,
		roster:     roster,
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan bool),
	}
	go b.loop()
	if disco != nil {
		disco.RegisterServerFeature(blockingCommandNamespace)
		disco.RegisterAccountFeature(blockingCommandNamespace)
	}
	return b, b.shutdownCh
}

// MatchesIQ returns whether or not an IQ should be
// processed by the blocking command module.
func (x *BlockingCommand) MatchesIQ(iq *xmpp.IQ) bool {
	e := iq.Elements()
	blockList := e.ChildNamespace("blocklist", blockingCommandNamespace)
	block := e.ChildNamespace("block", blockingCommandNamespace)
	unblock := e.ChildNamespace("unblock", blockingCommandNamespace)
	return (iq.IsGet() && blockList != nil) || (iq.IsSet() && (block != nil || unblock != nil))
}

// ProcessIQ processes a blocking command IQ
// taking according actions over the associated stream.
func (x *BlockingCommand) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	x.actorCh <- func() { x.processIQ(iq, stm) }
}

// runs on it's own goroutine
func (x *BlockingCommand) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case c := <-x.shutdownCh:
			c <- true
			return
		}
	}
}

func (x *BlockingCommand) processIQ(iq *xmpp.IQ, stm stream.C2S) {
	if iq.IsGet() {
		x.sendBlockList(iq, stm)
	} else if iq.IsSet() {
		e := iq.Elements()
		if block := e.ChildNamespace("block", blockingCommandNamespace); block != nil {
			x.block(iq, block, stm)
		} else if unblock := e.ChildNamespace("unblock", blockingCommandNamespace); unblock != nil {
			x.unblock(iq, unblock, stm)
		}
	}
}

func (x *BlockingCommand) sendBlockList(iq *xmpp.IQ, stm stream.C2S) {
	fromJID := iq.FromJID()
	blItms, err := storage.FetchBlockListItems(fromJID.Node())
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	blockList := xmpp.NewElementNamespace("blocklist", blockingCommandNamespace)
	for _, blItm := range blItms {
		itElem := xmpp.NewElementName("item")
		itElem.SetAttribute("jid", blItm.JID)
		blockList.AppendElement(itElem)
	}
	reply := iq.ResultIQ()
	reply.AppendElement(blockList)
	stm.SendElement(reply)

	stm.SetBool(xep191RequestedContextKey, true)
}

func (x *BlockingCommand) block(iq *xmpp.IQ, block xmpp.XElement, stm stream.C2S) {
	var bl []model.BlockListItem

	items := block.Elements().Children("item")
	if len(items) == 0 {
		stm.SendElement(iq.BadRequestError())
		return
	}
	jds, err := x.extractItemJIDs(items)
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.JidMalformedError())
		return
	}
	blItems, ris, err := x.fetchBlockListAndRosterItems(stm)
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	username := stm.Username()
	for _, j := range jds {
		if !x.isJIDInBlockList(j, blItems) {
			x.broadcastPresenceMatchingJID(j, ris, xmpp.UnavailableType, stm)
			bl = append(bl, model.BlockListItem{Username: username, JID: j.String()})
		}
	}
	if err := storage.InsertBlockListItems(bl); err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	x.router.ReloadBlockList(username)

	stm.SendElement(iq.ResultIQ())
	x.pushIQ(block, stm)
}

func (x *BlockingCommand) unblock(iq *xmpp.IQ, unblock xmpp.XElement, stm stream.C2S) {
	items := unblock.Elements().Children("item")
	jds, err := x.extractItemJIDs(items)
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.JidMalformedError())
		return
	}
	blItems, ris, err := x.fetchBlockListAndRosterItems(stm)
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	username := stm.Username()
	var bl []model.BlockListItem
	if len(jds) == 0 {
		for _, blItem := range blItems {
			j, _ := jid.NewWithString(blItem.JID, true)
			x.broadcastPresenceMatchingJID(j, ris, xmpp.AvailableType, stm)
		}
		bl = blItems

	} else {
		for _, j := range jds {
			if x.isJIDInBlockList(j, blItems) {
				x.broadcastPresenceMatchingJID(j, ris, xmpp.AvailableType, stm)
				bl = append(bl, model.BlockListItem{Username: username, JID: j.String()})
			}
		}
	}
	if err := storage.DeleteBlockListItems(bl); err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	x.router.ReloadBlockList(username)

	stm.SendElement(iq.ResultIQ())
	x.pushIQ(unblock, stm)
}

func (x *BlockingCommand) pushIQ(elem xmpp.XElement, stm stream.C2S) {
	stms := x.router.UserStreams(stm.Username())
	for _, stm := range stms {
		if !stm.GetBool(xep191RequestedContextKey) {
			continue
		}
		iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
		iq.AppendElement(elem)
		stm.SendElement(iq)
	}
}

func (x *BlockingCommand) broadcastPresenceMatchingJID(jid *jid.JID, ris []rostermodel.Item, presenceType string, stm stream.C2S) {
	if x.roster == nil {
		// roster disabled
		return
	}
	presences := x.roster.OnlinePresencesMatchingJID(jid)
	for _, presence := range presences {
		if !x.isSubscribedTo(presence.FromJID().ToBareJID(), ris) {
			continue
		}
		p := xmpp.NewPresence(presence.FromJID(), stm.JID().ToBareJID(), presenceType)
		if presenceType == xmpp.AvailableType {
			p.AppendElements(presence.Elements().All())
		}
		x.router.MustRoute(p)
	}
}

func (x *BlockingCommand) isJIDInBlockList(jid *jid.JID, blItems []model.BlockListItem) bool {
	for _, blItem := range blItems {
		if blItem.JID == jid.String() {
			return true
		}
	}
	return false
}

func (x *BlockingCommand) isSubscribedTo(jid *jid.JID, ris []rostermodel.Item) bool {
	str := jid.String()
	for _, ri := range ris {
		if ri.JID == str && (ri.Subscription == rostermodel.SubscriptionTo || ri.Subscription == rostermodel.SubscriptionBoth) {
			return true
		}
	}
	return false
}

func (x *BlockingCommand) fetchBlockListAndRosterItems(stm stream.C2S) ([]model.BlockListItem, []rostermodel.Item, error) {
	username := stm.Username()
	blItms, err := storage.FetchBlockListItems(username)
	if err != nil {
		return nil, nil, err
	}
	ris, _, err := storage.FetchRosterItems(username)
	if err != nil {
		return nil, nil, err
	}
	return blItms, ris, nil
}

func (x *BlockingCommand) extractItemJIDs(items []xmpp.XElement) ([]*jid.JID, error) {
	var ret []*jid.JID
	for _, item := range items {
		j, err := jid.NewWithString(item.Attributes().Get("jid"), false)
		if err != nil {
			return nil, err
		}
		ret = append(ret, j)
	}
	return ret, nil
}
