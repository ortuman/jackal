/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestItemElement(t *testing.T) {
	elem := xmpp.NewElementName("item2")
	it, err := NewItem(elem)
	require.Nil(t, it)
	require.NotNil(t, err)

	// no jid
	elem.SetName("item")
	it, err = NewItem(elem)
	require.Nil(t, it)
	require.NotNil(t, err)

	// bad jid
	elem.SetAttribute("jid", string([]byte{255, 255, 255}))
	it, err = NewItem(elem)
	require.Nil(t, it)
	require.NotNil(t, err)

	// bad subscription
	elem.SetAttribute("jid", "ortuman@jackal.im")
	elem.SetAttribute("subscription", "foo")
	it, err = NewItem(elem)
	require.Nil(t, it)
	require.NotNil(t, err)

	// bad ask
	elem.SetAttribute("subscription", "both")
	elem.SetAttribute("ask", "foo")
	it, err = NewItem(elem)
	require.Nil(t, it)
	require.NotNil(t, err)

	// attach bad group
	elem.SetAttribute("ask", "subscribe")
	elem.AppendElement(xmpp.NewElementNamespace("group", "ns"))
	it, err = NewItem(elem)
	require.Nil(t, it)
	require.NotNil(t, err)

	elem.RemoveElements("group")
	gr := xmpp.NewElementName("group")
	gr.SetText("friends")
	elem.AppendElement(gr)
	elem.SetAttribute("name", "buddy")
	it, err = NewItem(elem)
	require.NotNil(t, it)
	require.Nil(t, err)

	itElem := it.Element()
	require.Equal(t, "item", itElem.Name())
	require.Equal(t, "buddy", itElem.Attributes().Get("name"))
	require.Equal(t, "ortuman@jackal.im", itElem.Attributes().Get("jid"))
	require.Equal(t, "both", itElem.Attributes().Get("subscription"))
	require.Equal(t, "subscribe", itElem.Attributes().Get("ask"))
	require.Equal(t, 1, len(itElem.Elements().All()))
}

func TestItem_Serialize(t *testing.T) {
	var ri1 Item
	ri1 = Item{
		Username:     "ortuman",
		JID:          "noelia",
		Ask:          true,
		Subscription: "none",
		Groups:       []string{"friends", "family"},
	}
	buf := new(bytes.Buffer)
	ri1.ToGob(gob.NewEncoder(buf))
	ri2 := &Item{}
	ri2.FromGob(gob.NewDecoder(buf))
	require.Equal(t, ri1, *ri2)
}
