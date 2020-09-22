/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestXEP0045_CreateOwner(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	occJID, _ := jid.New("room", "conference.jackal.im", "nick", true)
	fullJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	o, err := muc.createOwner(nil, fullJID, occJID)
	require.Nil(t, err)

	oMem, err := muc.repOccupant.FetchOccupant(nil, occJID)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	assert.EqualValues(t, o, oMem)
}

func TestXEP0045_CreateOccupant(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	occJID, _ := jid.New("room", "conference.jackal.im", "nick", true)
	fullJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	o, err := muc.newOccupant(nil, fullJID, occJID)
	require.Nil(t, err)

	oMem, err := muc.repOccupant.FetchOccupant(nil, occJID)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	assert.EqualValues(t, o, oMem)

	errUsr, _ := jid.New("milos", "jackal.im", "laptop", true)
	errOcc, err := muc.newOccupant(nil, errUsr, occJID)
	require.NotNil(t, err)
	require.Nil(t, errOcc)
}
