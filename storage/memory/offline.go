/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/model/serializer"
	"github.com/ortuman/jackal/xmpp"
)

// Offline represents an in-memory offline storage.
type Offline struct {
	*memoryStorage
}

// NewOffline returns an instance of Offline in-memory storage.
func NewOffline() *Offline {
	return &Offline{memoryStorage: newStorage()}
}

// InsertOfflineMessage inserts a new message element into user's offline queue.
func (m *Offline) InsertOfflineMessage(_ context.Context, message *xmpp.Message, username string) error {
	return m.updateInWriteLock(offlineMessageKey(username), func(b []byte) ([]byte, error) {
		var messages []xmpp.Message
		if len(b) > 0 {
			if err := serializer.DeserializeSlice(b, &messages); err != nil {
				return nil, err
			}
		}
		messages = append(messages, *message)

		b, err := serializer.SerializeSlice(&messages)
		if err != nil {
			return nil, err
		}
		return b, nil
	})
}

// CountOfflineMessages returns current length of user's offline queue.
func (m *Offline) CountOfflineMessages(_ context.Context, username string) (int, error) {
	var messages []xmpp.Message
	_, err := m.getEntities(offlineMessageKey(username), &messages)
	if err != nil {
		return 0, err
	}
	return len(messages), nil
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func (m *Offline) FetchOfflineMessages(_ context.Context, username string) ([]xmpp.Message, error) {
	var messages []xmpp.Message
	_, err := m.getEntities(offlineMessageKey(username), &messages)
	switch err {
	case nil:
		return messages, nil
	default:
		return nil, err
	}
}

// DeleteOfflineMessages clears a user offline queue.
func (m *Offline) DeleteOfflineMessages(_ context.Context, username string) error {
	return m.deleteKey(offlineMessageKey(username))
}

func offlineMessageKey(username string) string {
	return "offlineMessages:" + username
}
