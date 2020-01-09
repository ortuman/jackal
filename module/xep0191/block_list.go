/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0191

import (
	"context"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/presencehub"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
)

const blockingCommandNamespace = "urn:xmpp:blocking"

const (
	xep191RequestedContextKey = "xep_191:requested"
)

// BlockingCommand represents a blocking command IQ handler module.
type BlockingCommand struct {
	runQueue     *runqueue.RunQueue
	router       *router.Router
	blockListRep repository.BlockList
	rosterRep    repository.Roster
	presenceHub  *presencehub.PresenceHub
}

// New returns a blocking command IQ handler module.
func New(disco *xep0030.DiscoInfo, presenceHub *presencehub.PresenceHub, router *router.Router, rosterRep repository.Roster, blockListRep repository.BlockList) *BlockingCommand {
	b := &BlockingCommand{
		runQueue:     runqueue.New("xep0191"),
		router:       router,
		blockListRep: blockListRep,
		rosterRep:    rosterRep,
		presenceHub:  presenceHub,
	}
	if disco != nil {
		disco.RegisterServerFeature(blockingCommandNamespace)
		disco.RegisterAccountFeature(blockingCommandNamespace)
	}
	return b
}

// MatchesIQ returns whether or not an IQ should be processed by the blocking command module.
func (x *BlockingCommand) MatchesIQ(iq *xmpp.IQ) bool {
	e := iq.Elements()
	blockList := e.ChildNamespace("blocklist", blockingCommandNamespace)
	block := e.ChildNamespace("block", blockingCommandNamespace)
	unblock := e.ChildNamespace("unblock", blockingCommandNamespace)
	return (iq.IsGet() && blockList != nil) || (iq.IsSet() && (block != nil || unblock != nil))
}

// ProcessIQ processes a blocking command IQ taking according actions over the associated stream.
func (x *BlockingCommand) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		stm := x.router.UserStream(iq.FromJID())
		if stm == nil {
			return
		}
		x.processIQ(ctx, iq, stm)
	})
}

// Shutdown shuts down blocking module.
func (x *BlockingCommand) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *BlockingCommand) processIQ(ctx context.Context, iq *xmpp.IQ, stm stream.C2S) {
	if iq.IsGet() {
		x.sendBlockList(ctx, iq, stm)
	} else if iq.IsSet() {
		e := iq.Elements()
		if block := e.ChildNamespace("block", blockingCommandNamespace); block != nil {
			x.block(ctx, iq, block, stm)
		} else if unblock := e.ChildNamespace("unblock", blockingCommandNamespace); unblock != nil {
			x.unblock(ctx, iq, unblock, stm)
		}
	}
}

func (x *BlockingCommand) sendBlockList(ctx context.Context, iq *xmpp.IQ, stm stream.C2S) {
	fromJID := iq.FromJID()
	blItems, err := x.blockListRep.FetchBlockListItems(ctx, fromJID.Node())
	if err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.InternalServerError())
		return
	}
	blockList := xmpp.NewElementNamespace("blocklist", blockingCommandNamespace)
	for _, blItem := range blItems {
		itElem := xmpp.NewElementName("item")
		itElem.SetAttribute("jid", blItem.JID)
		blockList.AppendElement(itElem)
	}
	stm.SetBool(ctx, xep191RequestedContextKey, true)

	reply := iq.ResultIQ()
	reply.AppendElement(blockList)
	stm.SendElement(ctx, reply)
}

func (x *BlockingCommand) block(ctx context.Context, iq *xmpp.IQ, block xmpp.XElement, stm stream.C2S) {
	items := block.Elements().Children("item")
	if len(items) == 0 {
		stm.SendElement(ctx, iq.BadRequestError())
		return
	}
	jds, err := x.extractItemJIDs(items)
	if err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.JidMalformedError())
		return
	}
	blItems, ris, err := x.fetchBlockListAndRosterItems(ctx, stm.Username())
	if err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.InternalServerError())
		return
	}
	username := stm.Username()
	for _, j := range jds {
		if !x.isJIDInBlockList(j, blItems) {
			err := x.blockListRep.InsertBlockListItem(ctx, &model.BlockListItem{
				Username: username,
				JID:      j.String(),
			})
			if err != nil {
				log.Error(err)
				stm.SendElement(ctx, iq.InternalServerError())
				return
			}
			x.broadcastPresenceMatchingJID(ctx, j, ris, xmpp.UnavailableType, stm)
		}
	}
	x.router.ReloadBlockList(username)

	stm.SendElement(ctx, iq.ResultIQ())
	x.pushIQ(ctx, block, stm)
}

func (x *BlockingCommand) unblock(ctx context.Context, iq *xmpp.IQ, unblock xmpp.XElement, stm stream.C2S) {
	items := unblock.Elements().Children("item")
	jds, err := x.extractItemJIDs(items)
	if err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.JidMalformedError())
		return
	}
	username := stm.Username()

	blItems, ris, err := x.fetchBlockListAndRosterItems(ctx, username)
	if err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.InternalServerError())
		return
	}
	if len(jds) > 0 {
		for _, j := range jds {
			if x.isJIDInBlockList(j, blItems) {
				if err := x.blockListRep.DeleteBlockListItem(ctx, &model.BlockListItem{
					Username: username,
					JID:      j.String(),
				}); err != nil {
					log.Error(err)
					stm.SendElement(ctx, iq.InternalServerError())
					return
				}
				x.broadcastPresenceMatchingJID(ctx, j, ris, xmpp.AvailableType, stm)
			}
		}
	} else { // remove all block list items
		for _, blItem := range blItems {
			if err := x.blockListRep.DeleteBlockListItem(ctx, &blItem); err != nil {
				log.Error(err)
				stm.SendElement(ctx, iq.InternalServerError())
				return
			}
			j, _ := jid.NewWithString(blItem.JID, true)
			x.broadcastPresenceMatchingJID(ctx, j, ris, xmpp.AvailableType, stm)
		}
	}
	x.router.ReloadBlockList(username)

	stm.SendElement(ctx, iq.ResultIQ())
	x.pushIQ(ctx, unblock, stm)
}

func (x *BlockingCommand) pushIQ(ctx context.Context, elem xmpp.XElement, stm stream.C2S) {
	streams := x.router.UserStreams(stm.Username())
	for _, stm := range streams {
		if !stm.GetBool(xep191RequestedContextKey) {
			continue
		}
		iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
		iq.AppendElement(elem)
		stm.SendElement(ctx, iq)
	}
}

func (x *BlockingCommand) broadcastPresenceMatchingJID(ctx context.Context, jid *jid.JID, ris []rostermodel.Item, presenceType string, stm stream.C2S) {
	if x.presenceHub == nil {
		// roster disabled
		return
	}
	onlinePresences := x.presenceHub.AvailablePresencesMatchingJID(jid)
	for _, onlinePresence := range onlinePresences {
		presence := onlinePresence.Presence
		if !x.isSubscribedTo(presence.FromJID().ToBareJID(), ris) {
			continue
		}
		p := xmpp.NewPresence(presence.FromJID(), stm.JID().ToBareJID(), presenceType)
		if presenceType == xmpp.AvailableType {
			p.AppendElements(presence.Elements().All())
		}
		_ = x.router.MustRoute(ctx, p)
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

func (x *BlockingCommand) fetchBlockListAndRosterItems(ctx context.Context, username string) ([]model.BlockListItem, []rostermodel.Item, error) {
	blItems, err := x.blockListRep.FetchBlockListItems(ctx, username)
	if err != nil {
		return nil, nil, err
	}
	ris, _, err := x.rosterRep.FetchRosterItems(ctx, username)
	if err != nil {
		return nil, nil, err
	}
	return blItems, ris, nil
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
