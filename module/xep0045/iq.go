/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
)

func isIQForKickOccupant(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	item := query.Elements().Child("item")
	if item == nil || item.Attributes().Get("nick") == "" ||
		item.Attributes().Get("role") != "none" {
		return false
	}
	return true
}

func (s *Muc) kickOccupant(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	mod, errStanza := s.getModeratorFromIQ(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	kickedOcc, err := s.getKickedOccupant(ctx, room, iq)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
		return
	}

	if !mod.CanKickOccupant(kickedOcc) {
		_ = s.router.Route(ctx, iq.NotAllowedError())
		return
	}

	kickedOcc.SetAffiliation("")
	kickedOcc.SetRole("")
	s.occupantExitsRoom(ctx, room, kickedOcc)

	reasonEl := iq.Elements().Child("query").Elements().Child("item").Elements().Child("reason")
	reason := ""
	if reasonEl != nil {
		reason = reasonEl.Text()
	}
	s.notifyKickedOccupant(ctx, kickedOcc, mod.OccupantJID.Resource(), reason)
	_ = s.router.Route(ctx, iq.ResultIQ())
	s.notifyRoomOccupantKicked(ctx, room, kickedOcc, mod.OccupantJID.Resource(), reason)
}

func (s *Muc) notifyKickedOccupant(ctx context.Context, o *mucmodel.Occupant, actor, reason string) {
	el := getKickedOccupantElement(actor, reason, true)
	for _, resource := range o.GetAllResources() {
		to := addResourceToBareJID(o.BareJID, resource)
		p, _ := xmpp.NewPresenceFromElement(el, o.OccupantJID, to)
		_ = s.router.Route(ctx, p)
	}
}

func (s *Muc) notifyRoomOccupantKicked(ctx context.Context, room *mucmodel.Room,
	kicked *mucmodel.Occupant, actor, reason string) {
	el := getKickedOccupantElement(actor, reason, false)
	for _, occJID := range room.GetAllOccupantJIDs() {
		o , _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		for _, resource := range o.GetAllResources() {
			to := addResourceToBareJID(o.BareJID, resource)
			p, _ := xmpp.NewPresenceFromElement(el, o.OccupantJID, to)
			_ = s.router.Route(ctx, p)
		}

	}
}

func (s *Muc) getKickedOccupant(ctx context.Context, room *mucmodel.Room,
	iq *xmpp.IQ) (*mucmodel.Occupant, error) {
	kickedOccNick := iq.Elements().Child("query").Elements().Child("item").Attributes().Get("nick")
	kickedOccJID := addResourceToBareJID(room.RoomJID, kickedOccNick)
	kickedOcc, err := s.repOccupant.FetchOccupant(ctx, kickedOccJID)
	if err != nil {
		return nil, err
	}
	if kickedOcc == nil {
		return nil, fmt.Errorf("Occupant %s does not exist", kickedOccJID.String())
	}
	return kickedOcc, nil
}

func isIQForInstantRoomCreate(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	x := query.Elements().Child("x")
	if x == nil {
		return false
	}
	if x.Namespace() != "jabber:x:data" || x.Type() != "submit" || x.Elements().Count() != 0 {
		return false
	}
	return true
}

func (s *Muc) createInstantRoom(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	_, errStanza := s.getOwnerFromIQ(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	room.Locked = false
	err := s.repRoom.UpsertRoom(ctx, room)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
	}

	_ = s.router.Route(ctx, iq.ResultIQ())
}

func isIQForRoomConfigRequest(iq *xmpp.IQ) bool {
	if !iq.IsGet() {
		return false
	}
	query := iq.Elements().Child("query")
	if query.Elements().Count() != 0 {
		return false
	}
	return true
}

func (s *Muc) sendRoomConfiguration(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	_, errStanza := s.getOwnerFromIQ(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	configForm := s.getRoomConfigForm(ctx, room)
	stanza := getFormStanza(iq, configForm)
	_ = s.router.Route(ctx, stanza)
}

func isIQForRoomConfigSubmission(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	form := query.Elements().Child("x")
	if form == nil || form.Namespace() != xep0004.FormNamespace || form.Type() != "submit" {
		return false
	}
	return true
}

func (s *Muc) processRoomConfiguration(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	_, errStanza := s.getOwnerFromIQ(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	formEl := iq.Elements().Child("query").Elements().Child("x")
	form, err := xep0004.NewFormFromElement(formEl)
	if err != nil {
		_ = s.router.Route(ctx, iq.BadRequestError())
		return
	}

	ok := s.updateRoomWithForm(ctx, room, form)
	if !ok {
		_ = s.router.Route(ctx, iq.NotAcceptableError())
		return
	}

	_ = s.router.Route(ctx, iq.ResultIQ())
}
