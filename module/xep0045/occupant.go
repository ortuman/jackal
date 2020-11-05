/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0004"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	voiceRequestInstructions = "To approve this voice request select the checkbox and click OK."
)

func (s *Muc) createOwner(ctx context.Context, userJID, occJID *jid.JID) (*mucmodel.Occupant, error) {
	o, err := s.newOccupant(ctx, userJID, occJID)
	if err != nil {
		return nil, err
	}
	o.SetAffiliation("owner")
	err = s.repOccupant.UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Muc) newOccupant(ctx context.Context, userJID, occJID *jid.JID) (*mucmodel.Occupant, error) {
	// check if the occupant already exists
	o, err := s.repOccupant.FetchOccupant(ctx, occJID)
	switch {
	case err != nil:
		return nil, err
	case o == nil:
		o, err = mucmodel.NewOccupant(occJID, userJID.ToBareJID())
		if err != nil {
			return nil, err
		}
	case userJID.ToBareJID().String() != o.BareJID.String():
		return nil, fmt.Errorf("xep0045: Can't use another user's occupant nick")
	}

	if !userJID.IsFull() {
		return nil, fmt.Errorf("xep0045: User jid has to specify the resource")

	}
	o.AddResource(userJID.Resource())

	err = s.repOccupant.UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Muc) getOccupantFromStanza(ctx context.Context, room *mucmodel.Room,
	stanza xmpp.Stanza) (*mucmodel.Occupant, xmpp.Stanza) {
	occJID, ok := room.GetOccupantJID(stanza.FromJID().ToBareJID())
	if !ok {
		return nil, xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrForbidden, nil)
	}

	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		return nil, xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrInternalServerError, nil)
	}
	return occ, nil
}

func (s *Muc) getOwnerFromIQ(ctx context.Context, room *mucmodel.Room,
	iq *xmpp.IQ) (*mucmodel.Occupant, xmpp.Stanza) {
	occ, errStanza := s.getOccupantFromStanza(ctx, room, iq)
	if errStanza != nil {
		return nil, errStanza
	}

	if !occ.IsOwner() {
		return nil, iq.ForbiddenError()
	}

	return occ, nil
}

func (s *Muc) getOccupantsByRole(ctx context.Context, room *mucmodel.Room,
	sender *mucmodel.Occupant, role string) ([]*mucmodel.Occupant, error) {
	if !sender.IsModerator() {
		return nil, fmt.Errorf("xep0045: only mods can retrive the list of %ss", role)
	}
	res := make([]*mucmodel.Occupant, 0)
	for _, occJID := range room.GetAllOccupantJIDs() {
		o, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		if o.GetRole() == role {
			res = append(res, o)
		}
	}
	return res, nil
}

func (s *Muc) getOccupantsByAffiliation(ctx context.Context, room *mucmodel.Room,
	sender *mucmodel.Occupant, aff string) ([]*mucmodel.Occupant, error) {
	switch aff {
	case "outcast", "member":
		if !sender.IsAdmin() && !sender.IsOwner() {
			return nil, fmt.Errorf("xep0045: only admins and owners can get %ss", aff)
		}
	case "owner", "admin":
		if !sender.IsOwner() {
			return nil, fmt.Errorf("xep0045: only owners can retrive the %ss", aff)
		}
	default:
		return nil, fmt.Errorf("xep0045: unknown affiliation")
	}

	res := make([]*mucmodel.Occupant, 0)
	for _, occJID := range room.GetAllOccupantJIDs() {
		o, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		if o.GetAffiliation() == aff {
			res = append(res, o)
		}
	}
	return res, nil
}

func (s *Muc) sendPresenceToOccupant(ctx context.Context, o *mucmodel.Occupant,
	from *jid.JID, presenceEl *xmpp.Element) error {
	for _, resource := range o.GetAllResources() {
		to := addResourceToBareJID(o.BareJID, resource)
		p, err := xmpp.NewPresenceFromElement(presenceEl, from, to)
		if err != nil {
			return err
		}
		err = s.router.Route(ctx, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Muc) sendMessageToOccupant(ctx context.Context, o *mucmodel.Occupant,
	from *jid.JID, messageEl *xmpp.Element) error {
	for _, resource := range o.GetAllResources() {
		to := addResourceToBareJID(o.BareJID, resource)
		message, err := xmpp.NewMessageFromElement(messageEl, from, to)
		if err != nil {
			return err
		}
		err = s.router.Route(ctx, message)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Muc) getVoiceRequestForm(ctx context.Context, o *mucmodel.Occupant) *xep0004.DataForm{
	form := &xep0004.DataForm{
		Type: xep0004.Form,
		Title: "Voice request",
		Instructions: voiceRequestInstructions,
	}
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    xep0004.FormType,
		Type:   xep0004.Hidden,
		Values: []string{"http://jabber.org/protocol/muc#request"},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    "muc#role",
		Type:   xep0004.ListSingle,
		Label: "Requested role",
		Values: []string{"participant"},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    "muc#jid",
		Type:   xep0004.JidSingle,
		Label: "User ID",
		Values: []string{o.BareJID.String()},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    "muc#roomnick",
		Type:   xep0004.TextSingle,
		Label: "Room nickname",
		Values: []string{o.OccupantJID.Resource()},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    "muc#request_allow",
		Type:   xep0004.Boolean,
		Label: "Grant voice to this person?",
		Values: []string{"false"},
	})
	return form
}
