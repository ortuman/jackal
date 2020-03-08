/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

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

type mySQLPresences struct {
	*mySQLStorage
	pool *pool.BufferPool
}

func newPresences(db *sql.DB) *mySQLPresences {
	return &mySQLPresences{
		mySQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

func (s *mySQLPresences) UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) (inserted bool, err error) {
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
		Columns("username", "domain", "resource", "presence", "node", "ver", "allocation_id", "updated_at", "created_at").
		Values(jid.Node(), jid.Domain(), jid.Resource(), rawXML, node, ver, allocationID, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE presence = ?, node = ?, ver = ?, allocation_id = ?, updated_at = NOW()", rawXML, node, ver, allocationID)
	stmRes, err := q.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return false, err
	}
	affectedRows, err := stmRes.RowsAffected()
	if err != nil {
		return false, err
	}
	return affectedRows == 1, nil
}

func (s *mySQLPresences) FetchPresence(ctx context.Context, jid *jid.JID) (*capsmodel.PresenceCaps, error) {
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

func (s *mySQLPresences) FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]capsmodel.PresenceCaps, error) {
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

func (s *mySQLPresences) DeletePresence(ctx context.Context, jid *jid.JID) error {
	_, err := sq.Delete("presences").
		Where(sq.And{
			sq.Eq{"username": jid.Node()},
			sq.Eq{"domain": jid.Domain()},
			sq.Eq{"resource": jid.Resource()},
		}).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLPresences) DeleteAllocationPresences(ctx context.Context, allocationID string) error {
	_, err := sq.Delete("presences").
		Where(sq.Eq{"allocation_id": allocationID}).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLPresences) ClearPresences(ctx context.Context) error {
	_, err := sq.Delete("presences").RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLPresences) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error {
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

func (s *mySQLPresences) FetchCapabilities(ctx context.Context, node, ver string) (*capsmodel.Capabilities, error) {
	var b string
	err := sq.Select("features").From("capabilities").
		Where(sq.And{sq.Eq{"node": node}, sq.Eq{"ver": ver}}).
		RunWith(s.db).QueryRowContext(ctx).Scan(&b)
	switch err {
	case nil:
		var caps capsmodel.Capabilities
		if err := json.NewDecoder(strings.NewReader(b)).Decode(&caps.Features); err != nil {
			return nil, err
		}
		return &caps, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
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
