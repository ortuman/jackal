/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"github.com/ortuman/jackal/model/serializer"
	"github.com/ortuman/jackal/xmpp"
)

// InsertOfflineMessage inserts a new message element into user's offline queue.
func (m *Storage) InsertOfflineMessage(message *xmpp.Message, username string) error {
	return m.inWriteLock(func() error {
		messages, err := m.fetchUserOfflineMessages(username)
		if err != nil {
			return err
		}
		messages = append(messages, *message)

		b, err := serializer.SerializeSlice(&messages)
		if err != nil {
			return err
		}
		m.bytes[offlineMessageKey(username)] = b
		return nil
	})
}

// CountOfflineMessages returns current length of user's offline queue.
func (m *Storage) CountOfflineMessages(username string) (int, error) {
	var messages []xmpp.Message
	if err := m.inReadLock(func() error {
		var fnErr error
		messages, fnErr = m.fetchUserOfflineMessages(username)
		return fnErr
	}); err != nil {
		return 0, err
	}
	return len(messages), nil
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func (m *Storage) FetchOfflineMessages(username string) ([]xmpp.Message, error) {
	var messages []xmpp.Message
	if err := m.inReadLock(func() error {
		var fnErr error
		messages, fnErr = m.fetchUserOfflineMessages(username)
		return fnErr
	}); err != nil {
		return nil, err
	}
	return messages, nil
}

// DeleteOfflineMessages clears a user offline queue.
func (m *Storage) DeleteOfflineMessages(username string) error {
	return m.inWriteLock(func() error {
		delete(m.bytes, offlineMessageKey(username))
		return nil
	})
}

func (m *Storage) fetchUserOfflineMessages(username string) ([]xmpp.Message, error) {
	b := m.bytes[offlineMessageKey(username)]
	if b == nil {
		return nil, nil
	}
	var messages []xmpp.Message
	if err := serializer.DeserializeSlice(b, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func offlineMessageKey(username string) string {
	return "offlineMessages:" + username
}
