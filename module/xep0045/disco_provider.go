/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	mucNamespace           = "http://jabber.org/protocol/muc"
	mucNamespaceUser       = "http://jabber.org/protocol/muc#user"
	mucNamespaceOwner      = "http://jabber.org/protocol/muc#owner"
	mucNamespaceAdmin      = "http://jabber.org/protocol/muc#admin"
	mucNamespaceStableID   = "http://jabber.org/protocol/muc#stable_id"
	mucNamespaceRoomConfig = "http://jabber.org/protocol/muc#roomconfig"

	mucHidden        = "muc_hidden"
	mucPublic        = "muc_public"
	mucMembersOnly   = "muc_membersonly"
	mucOpen          = "muc_open"
	mucModerated     = "muc_moderated"
	mucUnmoderated   = "muc_unmoderated"
	mucNonAnonymous  = "muc_nonanonymous"
	mucSemiAnonymous = "muc_nonanonymous"
	mucPwdProtected  = "muc_passwordprotected"
	mucUnsecured     = "muc_unsecured"
	mucPersistent    = "muc_persistent"
	mucTemporary     = "muc_temporary"

	mucUserItem = "x-roomuser-item"
)

type discoInfoProvider struct {
	service *Muc
}

func setupDiscoService(cfg *Config, disco *xep0030.DiscoInfo, mucService *Muc) {
	item := xep0030.Item{
		Jid:  cfg.MucHost,
		Name: cfg.Name,
	}
	disco.RegisterServerItem(item)
	disco.RegisterServerFeature(mucNamespace)

	provider := &discoInfoProvider{
		service: mucService,
	}
	disco.RegisterProvider(cfg.MucHost, provider)
}

func (p *discoInfoProvider) Identities(ctx context.Context, toJID, fromJID *jid.JID, node string) []xep0030.Identity {
	var identities []xep0030.Identity
	if toJID != nil && toJID.Node() != "" {
		room := p.getRoom(ctx, toJID)
		if node == "" {
			if room != nil {
				identities = append(identities, xep0030.Identity{Type: "text",
					Category: "conference", Name: room.Name})
			}
		} else if node == mucUserItem {
			if room != nil {
				occJID, ok := room.GetOccupantJID(fromJID.ToBareJID())
				if ok {
					identities = append(identities, xep0030.Identity{Type: "text",
						Category: "conference", Name: occJID.Resource()})
				}
			}
		}
	} else {
		identities = append(identities, xep0030.Identity{Type: "text", Category: "conference",
			Name: p.service.cfg.Name})
	}
	return identities
}

func (p *discoInfoProvider) Features(ctx context.Context, toJID, _ *jid.JID, _ string) ([]xep0030.Feature, *xmpp.StanzaError) {
	if toJID != nil && toJID.Node() != "" {
		return p.roomFeatures(ctx, toJID)
	} else {
		return []string{mucNamespace}, nil
	}
}

func (p *discoInfoProvider) Form(_ context.Context, _, _ *jid.JID, _ string) (*xep0004.DataForm, *xmpp.StanzaError) {
	return nil, nil
}

func (p *discoInfoProvider) Items(ctx context.Context, toJID, _ *jid.JID, _ string) ([]xep0030.Item, *xmpp.StanzaError) {
	if toJID != nil && toJID.Node() != "" {
		return p.roomOccupants(ctx, toJID)
	} else {
		return p.publicRooms(ctx)
	}
}

func (p *discoInfoProvider) roomOccupants(ctx context.Context, roomJID *jid.JID) ([]xep0030.Item, *xmpp.StanzaError) {
	var items []xep0030.Item
	room := p.getRoom(ctx, roomJID)
	if room == nil {
		return nil, xmpp.ErrItemNotFound
	}
	if room.Config.GetCanGetMemberList() == mucmodel.All {
		for _, occJID := range room.GetAllOccupantJIDs() {
			items = append(items, xep0030.Item{Jid: occJID.String()})
		}
	}
	return items, nil
}

func (p *discoInfoProvider) publicRooms(ctx context.Context) ([]xep0030.Item, *xmpp.StanzaError) {
	var items []xep0030.Item
	p.service.mu.Lock()
	for _, r := range p.service.allRooms {
		room := p.getRoom(ctx, &r)
		if room == nil {
			return nil, xmpp.ErrInternalServerError
		}
		if room.Config.Public && !room.Locked {
			item := xep0030.Item{
				Jid:  room.RoomJID.String(),
				Name: room.Name,
			}
			items = append(items, item)
		}
	}
	p.service.mu.Unlock()
	return items, nil
}

func (p *discoInfoProvider) roomFeatures(ctx context.Context, roomJID *jid.JID) ([]xep0030.Feature, *xmpp.StanzaError) {
	room := p.getRoom(ctx, roomJID)
	if room == nil {
		return nil, xmpp.ErrItemNotFound
	}

	features := getRoomFeatures(room)

	return features, nil
}

func (p *discoInfoProvider) getRoom(ctx context.Context, roomJID *jid.JID) *mucmodel.Room {
	r, err := p.service.repRoom.FetchRoom(ctx, roomJID)
	if err != nil {
		log.Error(err)
		return nil
	}
	return r
}

func getRoomFeatures(room *mucmodel.Room) []string {
	features := []string{mucNamespace, mucNamespaceStableID, mucNamespaceRoomConfig}

	if room.Config.Public {
		features = append(features, mucPublic)
	} else {
		features = append(features, mucHidden)
	}

	if room.Config.Open {
		features = append(features, mucOpen)
	} else {
		features = append(features, mucMembersOnly)
	}

	if room.Config.Moderated {
		features = append(features, mucModerated)
	} else {
		features = append(features, mucUnmoderated)
	}

	if room.Config.NonAnonymous {
		features = append(features, mucNonAnonymous)
	} else {
		features = append(features, mucSemiAnonymous)
	}

	if room.Config.PwdProtected {
		features = append(features, mucPwdProtected)
	} else {
		features = append(features, mucUnsecured)
	}

	if room.Config.Persistent {
		features = append(features, mucPersistent)
	} else {
		features = append(features, mucTemporary)
	}
	return features
}
