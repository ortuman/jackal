/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"strings"

	capsmodel "github.com/ortuman/jackal/model/capabilities"
	"github.com/ortuman/jackal/model/serializer"
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
func (m *Presences) UpsertPresence(_ context.Context, presence *xmpp.Presence, jid *jid.JID, _ string) (inserted bool, err error) {
	var ok bool
	k := presenceKey(jid)
	if err := m.inWriteLock(func() error {
		_, ok = m.b[k]
		b, err := serializer.Serialize(presence)
		if err != nil {
			return err
		}
		m.b[k] = b
		return nil
	}); err != nil {
		return false, err
	}
	return !ok, nil
}

// FetchPresence retrieves from storage a concrete registered presence.
func (m *Presences) FetchPresence(_ context.Context, jid *jid.JID) (*capsmodel.PresenceCaps, error) {
	var pCaps *capsmodel.PresenceCaps

	if err := m.inReadLock(func() error {
		b := m.b[presenceKey(jid)]
		if b == nil {
			return nil
		}
		presenceCaps, err := m.deserializePresence(b)
		if err != nil {
			return err
		}
		pCaps = presenceCaps
		return nil
	}); err != nil {
		return nil, err
	}
	return pCaps, nil
}

// FetchPresencesMatchingJID retrives all storage presences matching a certain JID
func (m *Presences) FetchPresencesMatchingJID(ctx context.Context, j *jid.JID) ([]capsmodel.PresenceCaps, error) {
	var usePrefix, useSuffix bool
	var res []capsmodel.PresenceCaps

	if j.IsFullWithUser() {
		pCaps, err := m.FetchPresence(ctx, j)
		if err != nil {
			return nil, err
		}
		if pCaps == nil {
			return nil, nil
		}
		return []capsmodel.PresenceCaps{*pCaps}, nil
	}
	usePrefix = j.IsBare()
	useSuffix = j.IsFullWithServer()

	if err := m.inReadLock(func() error {
		for k, v := range m.b {
			if !strings.HasPrefix(k, "presences:") {
				continue
			}
			kJID, _ := jid.NewWithString(k[10:], true)
			if usePrefix {
				if !j.MatchesWithOptions(kJID, jid.MatchesBare) {
					continue
				}
			} else if useSuffix {
				if !j.MatchesWithOptions(kJID, jid.MatchesDomain|jid.MatchesResource) {
					continue
				}
			} else if !j.MatchesWithOptions(kJID, jid.MatchesDomain) {
				continue
			}
			presenceCaps, err := m.deserializePresence(v)
			if err != nil {
				return err
			}
			res = append(res, *presenceCaps)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

// DeletePresence removes from storage a concrete registered presence.
func (m *Presences) DeletePresence(_ context.Context, jid *jid.JID) error {
	return m.deleteKey(presenceKey(jid))
}

func (m *Presences) DeleteAllocationPresences(ctx context.Context, _ string) error {
	return m.ClearPresences(ctx)
}

func (m *Presences) ClearPresences(_ context.Context) error {
	return m.inWriteLock(func() error {
		for k := range m.b {
			if !strings.HasPrefix(k, "presences:") {
				continue
			}
			delete(m.b, k)
		}
		return nil
	})
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

func (m *Presences) deserializePresence(b []byte) (*capsmodel.PresenceCaps, error) {
	var pCaps capsmodel.PresenceCaps
	var presence xmpp.Presence

	if err := serializer.Deserialize(b, &presence); err != nil {
		return nil, err
	}
	pCaps.Presence = &presence
	if c := presence.Capabilities(); c != nil {
		if capsB := m.b[capabilitiesKey(c.Node, c.Ver)]; capsB != nil {
			var caps capsmodel.Capabilities
			if err := serializer.Deserialize(capsB, &caps); err != nil {
				return nil, err
			}
			pCaps.Caps = &caps
		}
	}
	return &pCaps, nil
}

func presenceKey(jid *jid.JID) string {
	return "presences:" + jid.String()
}

func capabilitiesKey(node, ver string) string {
	return "capabilities:" + node + ":" + ver
}
