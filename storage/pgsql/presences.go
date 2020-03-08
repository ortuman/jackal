/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type pgSQLPresences struct {
	*pgSQLStorage
	pool *pool.BufferPool
}

func newPresences(db *sql.DB) *pgSQLPresences {
	return &pgSQLPresences{
		pgSQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

func (s *pgSQLPresences) UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) error {
	return nil
}

func (s *pgSQLPresences) FetchPresence(ctx context.Context, jid *jid.JID) (*xmpp.Presence, *model.Capabilities, error) {
	return nil, nil, nil
}

func (s *pgSQLPresences) FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]xmpp.Presence, []model.Capabilities, error) {
	return nil, nil, nil
}

func (s *pgSQLPresences) DeletePresence(ctx context.Context, jid *jid.JID) error {
	_, err := sq.Delete("presences").
		Where(sq.And{
			sq.Eq{"username": jid.Node()},
			sq.Eq{"domain": jid.Domain()},
			sq.Eq{"resource": jid.Resource()},
		}).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *pgSQLPresences) ClearAllocationPresences(ctx context.Context, allocationID string) error {
	_, err := sq.Delete("presences").
		Where(sq.Eq{"allocation_id": allocationID}).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *pgSQLPresences) UpsertCapabilities(ctx context.Context, caps *model.Capabilities) error {
	b, err := json.Marshal(caps.Features)
	if err != nil {
		return err
	}
	_, err = sq.Insert("capabilities").
		Columns("node", "ver", "features").
		Values(caps.Node, caps.Ver, b).
		Suffix("ON CONFLICT (node, ver) DO UPDATE SET features = $4", b).
		RunWith(s.db).ExecContext(ctx)
	return err
}
