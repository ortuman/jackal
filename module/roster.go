/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import "github.com/ortuman/jackal/xml"

const rosterNamespace = ""

type Roster struct {
}

func (r *Roster) AssociatedNamespaces() []string {
	return []string{}
}

func (r *Roster) MatchesIQ(iq *xml.IQ) bool {
	return iq.FindElementNamespace("query", rosterNamespace) != nil
}
