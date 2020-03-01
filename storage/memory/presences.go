/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type Presences struct {
	*memoryStorage
}

// NewPresences returns an instance of Presences in-memory storage.
func NewPresences() *Presences {
	return &Presences{memoryStorage: newStorage()}
}

// UpsertPresence inserts or updates a presence and links it to certain allocation.
func (m *Presences) UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) error {
	return nil
}

// FetchPresence retrieves from storage a concrete registered presence.
func (m *Presences) FetchPresence(ctx context.Context, jid *jid.JID) (*xmpp.Presence, *model.Capabilities, error) {
	return nil, nil, nil
}

// FetchPresencesMatchingJID retrives all storage presences matching a certain JID
func (m *Presences) FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]xmpp.Presence, []model.Capabilities, error) {
	return nil, nil, nil
}

// DeletePresence removes from storage a concrete registered presence.
func (m *Presences) DeletePresence(ctx context.Context, jid *jid.JID) error {
	return nil
}

// DeleteAllocationPresences removes from storage all presences associated to a given allocation.
func (m *Presences) DeleteAllocationPresences(ctx context.Context, allocationID string) error {
	return nil
}

// UpsertCapabilities inserts capabilities associated to a node+ver pair, or updates them if previously inserted..
func (m *Presences) UpsertCapabilities(ctx context.Context, caps *model.Capabilities) error {
	return nil
}
