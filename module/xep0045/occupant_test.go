/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_NewOccupantOwner(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	occJID, _ := jid.New("room", "conference.jackal.im", "nick", true)
	fullJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	o, err := muc.createOwner(nil, occJID, "nick", fullJID)
	require.Nil(t, err)

	oMem, err := c.Occupant().FetchOccupant(nil, occJID)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	require.Equal(t, o.FullJID.String(), oMem.FullJID.String())
}
