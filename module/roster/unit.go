/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

type rosterUnit struct {
}

func (ru *rosterUnit) pushRosterItem(ri *storage.RosterItem, to *xml.JID) {
	query := xml.NewElementNamespace("query", rosterNamespace)
	query.AppendElement(elementFromRosterItem(ri))

	streams := stream.C2S().AvailableStreams(to.Node())
	for _, strm := range streams {
		if !strm.IsRosterRequested() {
			continue
		}
		pushEl := xml.NewIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(strm.JID().ToFullJID())
		pushEl.AppendElement(query)
		strm.SendElement(pushEl)
	}
}
