/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	capsmodel "github.com/ortuman/jackal/model/capabilities"
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
func (m *Presences) UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) (inserted bool, err error) {
	return false, nil
}

// FetchPresence retrieves from storage a concrete registered presence.
func (m *Presences) FetchPresence(ctx context.Context, jid *jid.JID) (*capsmodel.PresenceCaps, error) {
	return nil, nil
}

// FetchPresencesMatchingJID retrives all storage presences matching a certain JID
func (m *Presences) FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]capsmodel.PresenceCaps, error) {
	return nil, nil
}

// DeletePresence removes from storage a concrete registered presence.
func (m *Presences) DeletePresence(ctx context.Context, jid *jid.JID) error {
	return nil
}

func (m *Presences) DeleteAllocationPresences(ctx context.Context, allocationID string) error {
	return nil
}

func (m *Presences) ClearPresences(ctx context.Context) error {
	return nil
}

func (m *Presences) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error {
	return nil
}
