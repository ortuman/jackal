/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"github.com/ortuman/jackal/model/serializer"
	"github.com/ortuman/jackal/xmpp"
)

// UpsertPrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func (m *Storage) UpsertPrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	var priv []xmpp.Element

	// convert to concrete type
	for _, el := range privateXML {
		priv = append(priv, *xmpp.NewElementFromElement(el))
	}
	b, err := serializer.SerializeSlice(&priv)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.bytes[privateStorageKey(username, namespace)] = b
		return nil
	})
}

// FetchPrivateXML retrieves from storage a private element.
func (m *Storage) FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[privateStorageKey(username, namespace)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var priv []xmpp.Element
	if err := serializer.DeserializeSlice(b, &priv); err != nil {
		return nil, err
	}
	var ret []xmpp.XElement
	for _, e := range priv {
		ret = append(ret, &e)
	}
	return ret, nil
}

func privateStorageKey(username, namespace string) string {
	return "privateElements:" + username + ":" + namespace
}
