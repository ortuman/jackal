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

// isIQForRoomDestroy returns true if iq stanza is for destroying a room
func isIQForRoomDestroy(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	destroy := query.Elements().Child("destroy")
	if destroy == nil {
		return false
	}
	return true
}

// destroyRoom proceses the iq aimed at destroying an existing muc room
func (s *Muc) destroyRoom(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	owner, errStanza := s.getOccupantFromStanza(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	// notify occupants in the room that the room is destroyed
	err := s.notifyRoomDestroyed(ctx, owner, room, iq)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
		return
	}

	s.deleteRoom(ctx, room)
	_ = s.router.Route(ctx, iq.ResultIQ())
}

func (s *Muc) notifyRoomDestroyed(ctx context.Context, owner *mucmodel.Occupant,
	room *mucmodel.Room, iq *xmpp.IQ) error {
	// the actor destroying the room is sent to all of the occupants in the item element
	owner.SetAffiliation("")
	owner.SetRole("")

	// create the stanza to notify the room
	itemEl := newOccupantItem(owner, false, false)
	destroyEl := iq.Elements().Child("query").Elements().Child("destroy")
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser)
	xEl.AppendElement(itemEl).AppendElement(destroyEl)
	presenceEl := xmpp.NewElementName("presence").SetType("unavailable").AppendElement(xEl)

	for _, occJID := range room.GetAllOccupantJIDs() {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			return err
		}
		err = s.sendPresenceToOccupant(ctx, o, o.OccupantJID, presenceEl)
		if err != nil {
			return err
		}
	}
	return nil
}

// modifyOccupantList handles the iq stanzas sent to the muc admin namespace of type set
func (s *Muc) modifyOccupantList(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	sender, errStanza := s.getOccupantFromStanza(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	query := iq.Elements().Child("query")
	items := query.Elements().Children("item")
	// one item per occupant whose privilege is being changed
	for _, item := range items {
		err := s.modifyOccupantPrivilege(ctx, room, sender, item)
		if err != nil {
			_ = s.router.Route(ctx, iq.BadRequestError())
			return
		}
	}

	_ = s.router.Route(ctx, iq.ResultIQ())
}

// modifyOccupantPrivilege changes occupants role/affiliation as specified in the item element
func (s *Muc) modifyOccupantPrivilege(ctx context.Context, room *mucmodel.Room,
	sender *mucmodel.Occupant, item xmpp.XElement) error {
	role := item.Attributes().Get("role")
	affiliation := item.Attributes().Get("affiliation")

	var err error
	switch {
	case role != "":
		err = s.modifyOccupantRole(ctx, room, sender, item)
	case affiliation != "":
		err = s.modifyOccupantAffiliation(ctx, room, sender, item)
	default:
		err = fmt.Errorf("Role and affiliation not specified")
	}
	return err
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

	reason := getReasonFromItem(item)
	if newRole == "none" {
		err := s.kickOccupant(ctx, room, occ, sender.OccupantJID.Resource(), reason)
		if err != nil {
			return err
		}
	} else {
		occ.SetRole(newRole)
		s.repOccupant.UpsertOccupant(ctx, occ)

		occEl := getOccupantChangeElement(occ, reason)
		err := s.sendPresenceToRoom(ctx, room, occ.OccupantJID, occEl)
		if err != nil {
			return err
		}
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
	actor, reason string) error {
	kickedOcc.SetAffiliation("")
	kickedOcc.SetRole("")
	s.occupantExitsRoom(ctx, room, kickedOcc)

	kickedElSelf := getKickedOccupantElement(actor, reason, true)
	err := s.sendPresenceToOccupant(ctx, kickedOcc, kickedOcc.OccupantJID, kickedElSelf)
	if err != nil {
		return err
	}

	kickedElRoom := getKickedOccupantElement(actor, reason, false)
	err = s.sendPresenceToRoom(ctx, room, kickedOcc.OccupantJID, kickedElRoom)
	if err != nil {
		return err
	}

	return nil
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

	reason := getReasonFromItem(item)
	occEl := getOccupantChangeElement(occ, reason)
	err := s.sendPresenceToRoom(ctx, room, occ.OccupantJID, occEl)
	if err != nil {
		return err
	}

	if newAffiliation == "none" || newAffiliation == "outcast" {
		err = s.handleUserRemoval(ctx, room, sender, occ, newAffiliation, reason)
		if err != nil {
			return err
		}
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

func (s *Muc) handleUserRemoval(ctx context.Context, room *mucmodel.Room, sender, occ *mucmodel.Occupant,
	newAffiliation, reason string) error {
	if !room.Config.Open && newAffiliation == "none" {
		removedEl := getRoomMemberRemovedElement(sender.OccupantJID.Resource(), reason)
		err := s.sendPresenceToRoom(ctx, room, occ.OccupantJID, removedEl)
		if err != nil {
			return err
		}
		room.OccupantLeft(occ)
		s.repOccupant.DeleteOccupant(ctx, occ.OccupantJID)
	} else if newAffiliation == "outcast" {
		bannedEl := getUserBannedElement(sender.OccupantJID.Resource(), reason)
		err := s.sendPresenceToRoom(ctx, room, occ.OccupantJID, bannedEl)
		if err != nil {
			return err
		}
	}
	return nil
}

// getOccupantList handles the iq stanzas sent to the muc admin namespace of type get
func (s *Muc) getOccupantList(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	sender, errStanza := s.getOccupantFromStanza(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	// resOccupants is the list of occupants that matches the role/affiliation from iq
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

// isIQForInstantRoomCreate returns true if iq stanza is for creating an instant room
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

// createInstantRoom unlocks the existing room specified in iq stanza
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

// isIQForRoomConfigRequest returns true if iq stanza is for retrieving a room configuration form
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

// sendRoomConfiguration returns the room configuration form to the roow owner who requested it
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

// isIQForRoomConfigSubmission returns true if iq stanza is for submitting a room configuration
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

// processRoomConfiguration handles the iq modifying the existing's room config
func (s *Muc) processRoomConfiguration(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	_, errStanza := s.getOwnerFromIQ(ctx, room, iq)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	formEl := iq.Elements().Child("query").Elements().Child("x")
	switch formEl.Type() {
	case "submit":
		errStanza := s.configureRoom(ctx, room, formEl, iq)
		if errStanza != nil {
			_ = s.router.Route(ctx, errStanza)
			return
		}
	case "cancel":
		if room.Locked {
			s.deleteRoom(ctx, room)
		}
	default:
		_ = s.router.Route(ctx, iq.BadRequestError())
		return
	}

	_ = s.router.Route(ctx, iq.ResultIQ())
}

func (s *Muc) configureRoom(ctx context.Context, room *mucmodel.Room, formEl xmpp.XElement,
	iq *xmpp.IQ) xmpp.Stanza {
	form, err := xep0004.NewFormFromElement(formEl)
	if err != nil {
		return iq.BadRequestError()
	}

	updatedAnonimity, ok := s.updateRoomWithForm(ctx, room, form)
	if !ok {
		return iq.NotAcceptableError()
	}

	updatedRoomEl := getRoomUpdatedElement(room.Config.NonAnonymous, updatedAnonimity)
	s.sendMessageToRoom(ctx, room, room.RoomJID, updatedRoomEl)
	return nil
}
