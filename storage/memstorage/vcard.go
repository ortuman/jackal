/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import "github.com/ortuman/jackal/xml"

// InsertOrUpdateVCard inserts a new vCard element into storage,
// or updates it in case it's been previously inserted.
func (m *Storage) InsertOrUpdateVCard(vCard xml.XElement, username string) error {
	return m.inWriteLock(func() error {
		m.vCards[username] = xml.NewElementFromElement(vCard)
		return nil
	})
}

// FetchVCard retrieves from storage a vCard element associated
// to a given user.
func (m *Storage) FetchVCard(username string) (xml.XElement, error) {
	var ret xml.XElement
	err := m.inReadLock(func() error {
		ret = m.vCards[username]
		return nil
	})
	return ret, err
}
