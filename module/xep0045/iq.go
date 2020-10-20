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

func isIQForInstantRoomCreate(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	if query == nil {
		return false
	}
	if query.Namespace() != mucNamespaceOwner || query.Elements().Count() != 1 {
		return false
	}
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
	if query == nil {
		return false
	}
	if query.Namespace() != mucNamespaceOwner || query.Elements().Count() != 0 {
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
	if query == nil {
		return false
	}
	if query.Namespace() != mucNamespaceOwner || query.Elements().Count() != 1 {
		return false
	}
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
