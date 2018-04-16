/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestModelUser(t *testing.T) {
	var usr1 User

	now := time.Now()
	usr1.Username = "ortuman"
	usr1.Password = "1234"
	usr1.LoggedOutStatus = "Gone!"
	usr1.LoggedOutAt = now

	buf := new(bytes.Buffer)
	usr1.ToGob(gob.NewEncoder(buf))
	usr2 := NewUserFromGob(gob.NewDecoder(buf))
	require.Equal(t, usr1.Username, usr2.Username)
	require.Equal(t, usr1.Password, usr2.Password)
	require.Equal(t, usr1.LoggedOutAt.Format(time.RFC3339), usr2.LoggedOutAt.Format(time.RFC3339))
}

func TestModelRosterItem(t *testing.T) {
	var ri1 RosterItem

	ri1 = RosterItem{
		User:         "ortuman",
		Contact:      "noelia",
		Ask:          true,
		Subscription: "none",
		Groups:       []string{"friends", "family"},
	}
	buf := new(bytes.Buffer)
	ri1.ToGob(gob.NewEncoder(buf))
	ri2 := NewRosterItemFromGob(gob.NewDecoder(buf))
	require.Equal(t, ri1, *ri2)
}

func TestModelRosterVersion(t *testing.T) {
	var rv1 RosterVersion

	rv1 = RosterVersion{Ver: 2, DeletionVer: 1}
	buf := new(bytes.Buffer)
	rv1.ToGob(gob.NewEncoder(buf))
	rv2 := NewRosterVersionFromGob(gob.NewDecoder(buf))
	require.Equal(t, rv1, rv2)
}

func TestModelRosterNotification(t *testing.T) {
	var rn1 RosterNotification

	rn1 = RosterNotification{
		User:     "ortuman",
		Contact:  "noelia",
		Elements: []xml.XElement{xml.NewElementNamespace("c", "http://jabber.org/protocol/caps")},
	}
	buf := new(bytes.Buffer)
	rn1.ToGob(gob.NewEncoder(buf))
	rn2 := NewRosterNotificationFromGob(gob.NewDecoder(buf))
	require.Equal(t, "ortuman", rn2.User)
	require.Equal(t, "noelia", rn2.Contact)
	require.Equal(t, 1, len(rn1.Elements))
	require.Equal(t, 1, len(rn2.Elements))
	require.Equal(t, rn1.Elements[0].String(), rn2.Elements[0].String())
}
