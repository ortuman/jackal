/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import "github.com/ortuman/jackal/xml"

type contactUnit struct {
	rosterUnit
	userUnit *userUnit
}

func (cu *contactUnit) receiveUserPresence(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) {
}
