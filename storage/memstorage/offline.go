/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import "github.com/ortuman/jackal/xml"

// InsertOfflineMessage inserts a new message element into
// user's offline queue.
func (m *Storage) InsertOfflineMessage(message xml.XElement, username string) error {
	return m.inWriteLock(func() error {
		msgs := m.offlineMessages[username]
		msgs = append(msgs, xml.NewElementFromElement(message))
		m.offlineMessages[username] = msgs
		return nil
	})
}

// CountOfflineMessages returns current length of user's offline queue.
func (m *Storage) CountOfflineMessages(username string) (int, error) {
	var ret int
	err := m.inReadLock(func() error {
		ret = len(m.offlineMessages[username])
		return nil
	})
	return ret, err
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func (m *Storage) FetchOfflineMessages(username string) ([]xml.XElement, error) {
	var ret []xml.XElement
	err := m.inReadLock(func() error {
		ret = m.offlineMessages[username]
		return nil
	})
	return ret, err
}

// DeleteOfflineMessages clears a user offline queue.
func (m *Storage) DeleteOfflineMessages(username string) error {
	return m.inWriteLock(func() error {
		delete(m.offlineMessages, username)
		return nil
	})
}
