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
	"github.com/ortuman/jackal/xmpp"
)

func isIQForRoleChange(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	item := query.Elements().Child("item")
	if item == nil || item.Attributes().Get("nick") == "" ||
		item.Attributes().Get("role") == "" {
		return false
	}
	return true
}

func (s *Muc) changeRole(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	sender, errStanza := s.getOccupantFromIQ(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	occ, role := s.getOccupantAndNewRole(ctx, room, iq)
	if occ == nil {
		_ = s.router.Route(ctx, iq.BadRequestError())
		return
	}

	if !sender.CanChangeRole(occ, role) {
		_ = s.router.Route(ctx, iq.NotAllowedError())
		return
	}

	occ.SetRole(role)
	s.repOccupant.UpsertOccupant(ctx, occ)

	_ = s.router.Route(ctx, iq.ResultIQ())
	s.notifyRoomRoleChange(ctx, room, occ, getReasonFromIQ(iq))
}

func (s *Muc) notifyRoomRoleChange(ctx context.Context, room *mucmodel.Room,
	occ *mucmodel.Occupant, reason string) {
	xEl := getOccupantRoleChangeElement(occ, reason)
	presenceEl := xmpp.NewElementName("presence").AppendElement(xEl)
	for _, occJID := range room.GetAllOccupantJIDs() {
		o, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		for _, resource := range o.GetAllResources() {
			to := addResourceToBareJID(o.BareJID, resource)
			p, _ := xmpp.NewPresenceFromElement(presenceEl, occ.OccupantJID, to)
			_ = s.router.Route(ctx, p)
		}
	}
}

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

	kickedOcc, newRole := s.getOccupantAndNewRole(ctx, room, iq)
	if kickedOcc == nil || newRole != "none" {
		_ = s.router.Route(ctx, iq.BadRequestError())
		return
	}

	if !mod.CanChangeRole(kickedOcc, newRole) {
		_ = s.router.Route(ctx, iq.NotAllowedError())
		return
	}

	kickedOcc.SetAffiliation("")
	kickedOcc.SetRole("")
	s.occupantExitsRoom(ctx, room, kickedOcc)

	reason := getReasonFromIQ(iq)
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
		o, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		for _, resource := range o.GetAllResources() {
			to := addResourceToBareJID(o.BareJID, resource)
			p, _ := xmpp.NewPresenceFromElement(el, o.OccupantJID, to)
			_ = s.router.Route(ctx, p)
		}

	}
}

func (s *Muc) getOccupantAndNewRole(ctx context.Context, room *mucmodel.Room,
	iq *xmpp.IQ) (*mucmodel.Occupant, string) {
	occNick := iq.Elements().Child("query").Elements().Child("item").Attributes().Get("nick")
	occJID := addResourceToBareJID(room.RoomJID, occNick)
	occ, err := s.repOccupant.FetchOccupant(ctx, occJID)
	if err != nil || occ == nil {
		return nil, ""
	}
	newRole := iq.Elements().Child("query").Elements().Child("item").Attributes().Get("role")
	return occ, newRole
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
