package repository

import (
	"context"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type Presences interface {
	// UpsertPresence inserts or updates a presence and links it to certain allocation.
	UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) error

	// FetchPresence retrieves from storage a concrete registered presence.
	FetchPresence(ctx context.Context, jid *jid.JID) (*xmpp.Presence, *model.Capabilities, error)

	// FetchPresencesMatchingJID retrives all storage presences matching a certain JID
	FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]xmpp.Presence, []model.Capabilities, error)

	// DeletePresence removes from storage a concrete registered presence.
	DeletePresence(ctx context.Context, jid *jid.JID) error

	// DeleteAllocationPresences removes from storage all presences associated to a given allocation.
	DeleteAllocationPresences(ctx context.Context, allocationID string) error

	// UpsertCapabilities inserts capabilities associated to a node+ver pair, or updates them if previously inserted..
	UpsertCapabilities(ctx context.Context, caps *model.Capabilities) error
}
