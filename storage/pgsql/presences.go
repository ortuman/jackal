/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	sq "github.com/Masterminds/squirrel"
	capsmodel "github.com/ortuman/jackal/model/capabilities"
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

func (s *pgSQLPresences) UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) (loaded bool, err error) {
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	if err := presence.ToXML(buf, true); err != nil {
		return false, err
	}
	var node, ver string
	if caps := presence.Capabilities(); caps != nil {
		node = caps.Node
		ver = caps.Ver
	}
	rawXML := buf.String()

	q := sq.Insert("presences").
		Columns("username", "domain", "resource", "presence", "node", "ver", "allocation_id").
		Values(jid.Node(), jid.Domain(), jid.Resource(), rawXML, node, ver, allocationID).
		Suffix("ON CONFLICT (username, domain, resource) DO UPDATE SET presence = $4, node = $5, ver = $6, allocation_id = $7").
		Suffix("RETURNING CASE WHEN updated_at=created_at THEN true ELSE false END AS inserted")

	var inserted bool
	err = q.RunWith(s.db).QueryRowContext(ctx).Scan(&inserted)
	if err != nil {
		return false, err
	}
	return inserted, nil
}

func (s *pgSQLPresences) FetchPresence(ctx context.Context, jid *jid.JID) (*capsmodel.PresenceCaps, error) {
	var rawXML, node, ver, featuresJSON string

	q := sq.Select("presence", "c.node", "c.ver", "c.features").
		From("presences AS p, capabilities AS c").
		Where(sq.And{
			sq.Eq{"username": jid.Node()},
			sq.Eq{"domain": jid.Domain()},
			sq.Eq{"resource": jid.Resource()},
			sq.Expr("p.node = c.node"),
			sq.Expr("p.ver = c.ver"),
		}).
		RunWith(s.db)

	err := q.ScanContext(ctx, &rawXML, &node, &ver, &featuresJSON)
	switch err {
	case nil:
		return scanPresenceAndCapabilties(rawXML, node, ver, featuresJSON)
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *pgSQLPresences) FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]capsmodel.PresenceCaps, error) {
	var preds sq.And
	if len(jid.Node()) > 0 {
		preds = append(preds, sq.Eq{"username": jid.Node()})
	}
	if len(jid.Domain()) > 0 {
		preds = append(preds, sq.Eq{"domain": jid.Domain()})
	}
	if len(jid.Resource()) > 0 {
		preds = append(preds, sq.Eq{"resource": jid.Resource()})
	}
	preds = append(preds, sq.Expr("p.node = c.node"))
	preds = append(preds, sq.Expr("p.ver = c.ver"))

	q := sq.Select("presence", "c.node", "c.ver", "c.features").
		From("presences AS p, capabilities AS c").
		Where(preds).
		RunWith(s.db)

	rows, err := q.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var res []capsmodel.PresenceCaps
	for rows.Next() {
		var rawXML, node, ver, featuresJSON string

		if err := rows.Scan(&rawXML, &node, &ver, &featuresJSON); err != nil {
			return nil, err
		}
		presenceCaps, err := scanPresenceAndCapabilties(rawXML, node, ver, featuresJSON)
		if err != nil {
			return nil, err
		}
		res = append(res, *presenceCaps)
	}
	return res, nil
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

func (s *pgSQLPresences) DeleteAllocationPresences(ctx context.Context, allocationID string) error {
	_, err := sq.Delete("presences").
		Where(sq.Eq{"allocation_id": allocationID}).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *pgSQLPresences) ClearPresences(ctx context.Context) error {
	_, err := sq.Delete("presences").RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *pgSQLPresences) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error {
	b, err := json.Marshal(caps.Features)
	if err != nil {
		return err
	}
	_, err = sq.Insert("capabilities").
		Columns("node", "ver", "features", "updated_at", "created_at").
		Values(caps.Node, caps.Ver, b, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE features = ?, updated_at = NOW()", b).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func scanPresenceAndCapabilties(rawXML, node, ver, featuresJSON string) (*capsmodel.PresenceCaps, error) {
	parser := xmpp.NewParser(strings.NewReader(rawXML), xmpp.DefaultMode, 0)
	elem, err := parser.ParseElement()
	if err != nil {
		return nil, err
	}
	fromJID, _ := jid.NewWithString(elem.From(), true)
	toJID, _ := jid.NewWithString(elem.To(), true)

	presence, err := xmpp.NewPresenceFromElement(elem, fromJID, toJID)
	if err != nil {
		return nil, err
	}
	var res capsmodel.PresenceCaps

	res.Presence = presence
	if len(featuresJSON) > 0 {
		res.Caps = &capsmodel.Capabilities{
			Node: node,
			Ver:  ver,
		}

		if err := json.NewDecoder(strings.NewReader(featuresJSON)).Decode(&res.Caps.Features); err != nil {
			return nil, err
		}
	}
	return &res, nil
}
