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
	"github.com/ortuman/jackal/model"
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

func (s *mySQLPresences) UpsertPresence(ctx context.Context, presence *xmpp.Presence, jid *jid.JID, allocationID string) error {
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	if err := presence.ToXML(buf, true); err != nil {
		return err
	}
	var node, ver string
	if caps := presence.Capabilities(); caps != nil {
		node = caps.Node
		ver = caps.Ver
	}
	rawXML := buf.String()

	q := sq.Insert("presences").
		Columns("username", "domain", "resource", "presence", "allocation_id", "node", "ver", "updated_at", "created_at").
		Values(jid.Node(), jid.Domain(), jid.Resource(), allocationID, rawXML, node, ver, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE presence = ?, node = ?, ver = ?, allocation_id = ?, updated_at = NOW()", rawXML, node, ver, allocationID)
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLPresences) FetchPresence(ctx context.Context, jid *jid.JID) (*xmpp.Presence, *model.Capabilities, error) {
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
		return nil, nil, nil
	default:
		return nil, nil, err
	}
}

func (s *mySQLPresences) FetchPresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]xmpp.Presence, []model.Capabilities, error) {
	return nil, nil, nil
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

func (s *mySQLPresences) UpsertCapabilities(ctx context.Context, caps *model.Capabilities) error {
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

func scanPresenceAndCapabilties(rawXML, node, ver, featuresJSON string) (*xmpp.Presence, *model.Capabilities, error) {
	parser := xmpp.NewParser(strings.NewReader(rawXML), xmpp.DefaultMode, 0)
	elem, err := parser.ParseElement()
	if err != nil {
		return nil, nil, err
	}
	fromJID, _ := jid.NewWithString(elem.From(), true)
	toJID, _ := jid.NewWithString(elem.To(), true)

	presence, err := xmpp.NewPresenceFromElement(elem, fromJID, toJID)
	if err != nil {
		return nil, nil, err
	}
	var caps *model.Capabilities
	if len(featuresJSON) > 0 {
		caps = &model.Capabilities{
			Node: node,
			Ver:  ver,
		}
		if err := json.NewDecoder(strings.NewReader(featuresJSON)).Decode(&caps.Features); err != nil {
			return nil, nil, err
		}
	}
	return presence, caps, nil
}
