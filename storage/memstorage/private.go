/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import "github.com/ortuman/jackal/xml"

// InsertOrUpdatePrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func (m *Storage) InsertOrUpdatePrivateXML(privateXML []xml.XElement, namespace string, username string) error {
	return m.inWriteLock(func() error {
		var elems []xml.XElement
		for _, prv := range privateXML {
			elems = append(elems, xml.NewElementFromElement(prv))
		}
		m.privateXML[username+":"+namespace] = elems
		return nil
	})
}

// FetchPrivateXML retrieves from storage a private element.
func (m *Storage) FetchPrivateXML(namespace string, username string) ([]xml.XElement, error) {
	var ret []xml.XElement
	err := m.inReadLock(func() error {
		ret = m.privateXML[username+":"+namespace]
		return nil
	})
	return ret, err
}
