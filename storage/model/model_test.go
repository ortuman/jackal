/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestModelUser(t *testing.T) {
	var usr1, usr2 User

	usr1.Username = "ortuman"
	usr1.Password = "1234"

	buf := new(bytes.Buffer)
	usr1.ToBytes(buf)
	usr2.FromBytes(buf)
	require.Equal(t, usr1, usr2)
}

func TestModelRosterItem(t *testing.T) {
	var ri1, ri2 RosterItem

	ri1 = RosterItem{
		User:         "ortuman",
		Contact:      "noelia",
		Ask:          true,
		Subscription: "none",
		Groups:       []string{"friends", "family"},
	}
	buf := new(bytes.Buffer)
	ri1.ToBytes(buf)
	ri2.FromBytes(buf)
	require.Equal(t, ri1, ri2)
}

func TestModelRosterNotification(t *testing.T) {
	var rn1, rn2 RosterNotification

	rn1 = RosterNotification{
		User:     "ortuman",
		Contact:  "noelia",
		Elements: []xml.Element{xml.NewElementNamespace("c", "http://jabber.org/protocol/caps")},
	}
	buf := new(bytes.Buffer)
	rn1.ToBytes(buf)
	rn2.FromBytes(buf)
	require.Equal(t, rn1, rn2)
}
