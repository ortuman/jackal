/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package muc

import (
	"context"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// TODO possible add other namespaces (#owner, #user etc.)
const mucNamespace = "http://jabber.org/protocol/muc"

var mucFeature = []string{
	"http://jabber.org/protocol/muc",
}

// TODO declare different room features once they are added
var roomFeature = []string{
	"add_room_features (section 6.4)",
}

type discoInfoProvider struct {
	roomRep repository.Room
	service *Service
}

func setupDiscoMuc(cfg *Config, disco *xep0030.DiscoInfo, mucService *Service) {
	item := xep0030.Item{
		Jid:  cfg.MucHost,
		Name: "Chatroom Service",
	}
	disco.RegisterServerItem(item)
	disco.RegisterServerFeature(mucNamespace)

	provider := &discoInfoProvider{
		service: mucService,
	}
	disco.RegisterProvider(cfg.MucHost, provider)
}

func (p *discoInfoProvider) Identities(ctx context.Context, toJID, _ *jid.JID, _ string) []xep0030.Identity {
	var identities []xep0030.Identity
	if len(toJID.Node()) > 0 {
		roomJID := toJID
		room := p.getRoom(ctx, roomJID)
		if room != nil {
			// TODO replace room.Name with room.Description once it is added
			identities = append(identities, xep0030.Identity{Type: "text", Category: "conference",
				Name: room.Name})
		}
	} else {
		identities = append(identities, xep0030.Identity{Type: "text", Category: "conference",
			Name: "Chat Service"})
	}
	return identities
}

func (p *discoInfoProvider) Features(_ context.Context, toJID, fromJID *jid.JID, _ string) ([]xep0030.Feature, *xmpp.StanzaError) {
	if len(toJID.Node()) > 0 {
		// TODO to be changed once the room features are added
		return roomFeature, nil
	} else {
		return mucFeature, nil
	}
}

func (p *discoInfoProvider) Form(_ context.Context, _, _ *jid.JID, _ string) (*xep0004.DataForm, *xmpp.StanzaError) {
	return nil, nil
}

func (p *discoInfoProvider) Items(ctx context.Context, toJID, fromJID *jid.JID, node string) ([]xep0030.Item, *xmpp.StanzaError) {
	if len(toJID.Node()) > 0 {
		return p.roomOccupants(ctx, toJID.Node())
	}
	return p.allRooms(ctx)
}

func (p *discoInfoProvider) roomOccupants(ctx context.Context, roomName string) ([]xep0030.Item, *xmpp.StanzaError) {
	// TODO implement this function as shown in Section 6.5 once occupants are added
	var items []xep0030.Item
	return items, nil
}

func (p *discoInfoProvider) allRooms(ctx context.Context) ([]xep0030.Item, *xmpp.StanzaError) {
	// TODO return all of the rooms as described in Section 6.3
	var items []xep0030.Item
	for _, r := range p.service.publicRooms {
		item := xep0030.Item{
			Jid:  r.RoomJID.String(),
			Name: r.Desc,
		}
		items = append(items, item)
	}
	return items, nil
}

func (p *discoInfoProvider) getRoom(ctx context.Context, roomJID *jid.JID) *mucmodel.Room {
	r, err := p.roomRep.FetchRoom(ctx, roomJID)
	if err != nil {
		log.Error(err)
		return nil
	}
	return r
}
