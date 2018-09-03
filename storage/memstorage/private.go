/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import "github.com/ortuman/jackal/xmpp"

// InsertOrUpdatePrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func (m *Storage) InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	return m.inWriteLock(func() error {
		var elems []xmpp.XElement
		for _, prv := range privateXML {
			elems = append(elems, xmpp.NewElementFromElement(prv))
		}
		m.privateXML[username+":"+namespace] = elems
		return nil
	})
}

// FetchPrivateXML retrieves from storage a private element.
func (m *Storage) FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	var ret []xmpp.XElement
	err := m.inReadLock(func() error {
		ret = m.privateXML[username+":"+namespace]
		return nil
	})
	return ret, err
}
