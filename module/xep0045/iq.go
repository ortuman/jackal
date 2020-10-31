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
	"github.com/ortuman/jackal/xmpp/jid"
)

func (s *Muc) modifyOccupantList(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	sender, errStanza := s.getOccupantFromStanza(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	query := iq.Elements().Child("query")
	items := query.Elements().Children("item")
	for _, item := range items {
		err := s.modifyOccupantPrivilege(ctx, room, sender, item)
		if err != nil {
			_ = s.router.Route(ctx, iq.BadRequestError())
			return
		}
	}

	_ = s.router.Route(ctx, iq.ResultIQ())
}

func (s *Muc) modifyOccupantPrivilege(ctx context.Context, room *mucmodel.Room,
	sender *mucmodel.Occupant, item xmpp.XElement) error {
	role := item.Attributes().Get("role")
	affiliation := item.Attributes().Get("affiliation")
	if role != "" {
		err := s.modifyOccupantRole(ctx, room, sender, item)
		return err
	} else if affiliation != "" {
		err := s.modifyOccupantAffiliation(ctx, room, sender, item)
		return err
	} else {
		return fmt.Errorf("Role and affiliation not specified")
	}
}

func (s *Muc) modifyOccupantRole(ctx context.Context, room *mucmodel.Room,
	sender *mucmodel.Occupant, item xmpp.XElement) error {
	occ, newRole := s.getOccupantAndNewRole(ctx, room, item)
	if occ == nil {
		return fmt.Errorf("Occupant not in the room")
	}

	if !sender.CanChangeRole(occ, newRole) {
		return fmt.Errorf("Sender not allowed to change the role")
	}

	if newRole == "none" {
		s.kickOccupant(ctx, room, occ, sender.OccupantJID.Resource(), getReasonFromItem(item))
	} else {
		occ.SetRole(newRole)
		s.repOccupant.UpsertOccupant(ctx, occ)
		s.notifyRoomOccupantChange(ctx, room, occ, getReasonFromItem(item))
	}

	return nil
}

func (s *Muc) getOccupantAndNewRole(ctx context.Context, room *mucmodel.Room,
	item xmpp.XElement) (*mucmodel.Occupant, string) {
	occNick := item.Attributes().Get("nick")
	occJID := addResourceToBareJID(room.RoomJID, occNick)
	occ, err := s.repOccupant.FetchOccupant(ctx, occJID)
	if err != nil || occ == nil {
		return nil, ""
	}
	newRole := item.Attributes().Get("role")
	return occ, newRole
}

func (s *Muc) kickOccupant(ctx context.Context, room *mucmodel.Room, kickedOcc *mucmodel.Occupant,
	actor, reason string) {
	kickedOcc.SetAffiliation("")
	kickedOcc.SetRole("")
	s.occupantExitsRoom(ctx, room, kickedOcc)

	s.notifyKickedOccupant(ctx, kickedOcc, actor, reason)
	s.notifyRoomOccupantKicked(ctx, room, kickedOcc, actor, reason)
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

func (s *Muc) notifyRoomOccupantChange(ctx context.Context, room *mucmodel.Room,
	occ *mucmodel.Occupant, reason string) {
	xEl := getOccupantChangeElement(occ, reason)
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

func (s *Muc) modifyOccupantAffiliation(ctx context.Context, room *mucmodel.Room,
	sender *mucmodel.Occupant, item xmpp.XElement) error {
	occ, newAffiliation := s.getOccupantAndNewAffiliation(ctx, room, item)
	if occ == nil {
		return fmt.Errorf("Occupant not in the room")
	}

	if !sender.CanChangeAffiliation(occ, newAffiliation) {
		return fmt.Errorf("Sender not allowed to change the affiliation")
	}

	occ.SetAffiliation(newAffiliation)
	room.SetDefaultRole(occ)
	s.repOccupant.UpsertOccupant(ctx, occ)

	s.notifyRoomOccupantChange(ctx, room, occ, getReasonFromItem(item))

	senderNick := sender.OccupantJID.Resource()
	reason := getReasonFromItem(item)
	if !room.Config.Open && newAffiliation == "none" {
		s.notifyRoomMemberRemoved(ctx, room, occ.OccupantJID, senderNick, reason)
		room.OccupantLeft(occ)
		s.repOccupant.DeleteOccupant(ctx, occ.OccupantJID)
	} else if newAffiliation == "outcast" {
		s.notifyRoomUserBanned(ctx, room, occ.OccupantJID, senderNick, reason)
	}

	return nil
}

func (s *Muc) getOccupantAndNewAffiliation(ctx context.Context, room *mucmodel.Room,
	item xmpp.XElement) (*mucmodel.Occupant, string) {
	userBareJIDStr := item.Attributes().Get("jid")
	userBareJID, err := jid.NewWithString(userBareJIDStr, true)
	if err != nil {
		return nil, ""
	}
	occJID, ok := room.GetOccupantJID(userBareJID)
	if !ok {
		return nil, ""
	}
	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil || occ == nil {
		return nil, ""
	}
	newAff := item.Attributes().Get("affiliation")
	return occ, newAff
}

func (s *Muc) notifyRoomMemberRemoved(ctx context.Context, room *mucmodel.Room, from *jid.JID,
	actor, reason string) {
	presenceEl := getRoomMemberRemovedElement(actor, reason)
	for _, occJID := range room.GetAllOccupantJIDs() {
		o, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		for _, resource := range o.GetAllResources() {
			to := addResourceToBareJID(o.BareJID, resource)
			p, _ := xmpp.NewPresenceFromElement(presenceEl, from, to)
			_ = s.router.Route(ctx, p)
		}
	}
}

func (s *Muc) notifyRoomUserBanned(ctx context.Context, room *mucmodel.Room, from *jid.JID,
	actor, reason string) {
	presenceEl := getUserBannedElement(actor, reason)
	for _, occJID := range room.GetAllOccupantJIDs() {
		o, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		for _, resource := range o.GetAllResources() {
			to := addResourceToBareJID(o.BareJID, resource)
			p, _ := xmpp.NewPresenceFromElement(presenceEl, from, to)
			_ = s.router.Route(ctx, p)
		}
	}
}

func (s *Muc) getOccupantList(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	sender, errStanza := s.getOccupantFromStanza(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	resOccupants, errStanza := s.getRequestedOccupants(ctx, room, sender, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	listEl := getOccupantsInfoElement(resOccupants, iq.ID(),
		room.Config.OccupantCanDiscoverRealJID(sender))
	iqRes, _ := xmpp.NewIQFromElement(listEl, room.RoomJID, iq.FromJID())
	_ = s.router.Route(ctx, iqRes)
}

func (s *Muc) getRequestedOccupants(ctx context.Context, room *mucmodel.Room,
	sender *mucmodel.Occupant, iq *xmpp.IQ) ([]*mucmodel.Occupant, xmpp.Stanza) {
	switch filter := getFilterFromIQ(iq); filter {
	case "moderator", "participant", "visitor":
		resOccupants, err := s.getOccupantsByRole(ctx, room, sender, filter)
		if err != nil {
			return nil, iq.NotAllowedError()
		}
		return resOccupants, nil
	case "owner", "admin", "member", "outcast":
		resOccupants, err := s.getOccupantsByAffiliation(ctx, room, sender, filter)
		if err != nil {
			return nil, iq.NotAllowedError()
		}
		return resOccupants, nil
	}

	return nil, iq.BadRequestError()
}

func getFilterFromIQ(iq *xmpp.IQ) string {
	item := iq.Elements().Child("query").Elements().Child("item")
	if item == nil {
		return ""
	}
	aff := item.Attributes().Get("affiliation")
	if aff != "" {
		return aff
	}
	return item.Attributes().Get("role")
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
