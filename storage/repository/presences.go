package repository

import (
	"context"

	capsmodel "github.com/ortuman/jackal/model/capabilities"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type Presences interface {
	// UpsertPresence inserts or updates a presence and links it to certain allocation.
	// On insertion 'inserted' return parameter will be true.
	UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) (inserted bool, err error)

	// FetchPresence retrieves from storage a previously registered presence.
	FetchPresence(ctx context.Context, jid *jid.JID) (*capsmodel.PresenceCaps, error)

	// FetchPresencesMatchingJID retrives all storage presences matching a certain JID
	FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]capsmodel.PresenceCaps, error)

	// DeletePresence removes from storage a concrete registered presence.
	DeletePresence(ctx context.Context, jid *jid.JID) error

	// DeleteAllocationPresences removes from storage all presences associated to a given allocation.
	DeleteAllocationPresences(ctx context.Context, allocationID string) error

	// ClearPresences wipes out all storage presences.
	ClearPresences(ctx context.Context) error

	// UpsertCapabilities inserts capabilities associated to a node+ver pair, or updates them if previously inserted..
	UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error
}
