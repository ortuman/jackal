/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"strings"

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
func (m *Presences) UpsertPresence(_ context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) (inserted bool, err error) {
	k := presenceKey(jid.Node(), jid.Domain(), jid.Resource(), allocationID)
	m.mu.RLock()
	_, ok := m.b[k]
	m.mu.RUnlock()

	return !ok, m.saveEntity(k, presence)
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

func (m *Presences) DeleteAllocationPresences(_ context.Context, allocationID string) error {
	m.mu.Lock()
	for k := range m.b {
		if !strings.HasSuffix(allocationID, k) {
			continue
		}
		delete(m.b, k)
	}
	m.mu.Unlock()
	return nil
}

func (m *Presences) ClearPresences(_ context.Context) error {
	m.mu.Lock()
	for k := range m.b {
		if !strings.HasPrefix("presences:", k) {
			continue
		}
		delete(m.b, k)
	}
	m.mu.Unlock()
	return nil
}

func (m *Presences) UpsertCapabilities(_ context.Context, caps *capsmodel.Capabilities) error {
	return m.saveEntity(capabilitiesKey(caps.Node, caps.Ver), caps)
}

func (m *Presences) FetchCapabilities(_ context.Context, node, ver string) (*capsmodel.Capabilities, error) {
	var caps capsmodel.Capabilities

	ok, err := m.getEntity(capabilitiesKey(node, ver), &caps)
	switch err {
	case nil:
		if !ok {
			return nil, nil
		}
		return &caps, nil
	default:
		return nil, err
	}
}

func presenceKey(username, domain, resource, allocationID string) string {
	return "presences:" + username + ":" + domain + ":" + resource + ":" + allocationID
}

func capabilitiesKey(node, ver string) string {
	return "capabilities:" + node + ":" + ver
}
