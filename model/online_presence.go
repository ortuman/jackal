package model

import (
	"github.com/ortuman/jackal/xmpp"
)

// OnlinePresence represents an available presence
type OnlinePresence struct {
	Presence *xmpp.Presence
	Caps     *Capabilities
}
