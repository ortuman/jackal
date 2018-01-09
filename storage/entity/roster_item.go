/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package entity

import (
	"github.com/ortuman/jackal/xml"
)

type RosterItem struct {
	Username     string
	JID          *xml.JID
	Name         string
	Subscription string
	Ask          bool
	Groups       []string
}
