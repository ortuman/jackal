// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgsqlrepository

import (
	"context"
	"database/sql"

	"github.com/jackal-xmpp/stravaganza"

	sq "github.com/Masterminds/squirrel"
	kitlog "github.com/go-kit/log"
	"github.com/golang/protobuf/proto"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
)

const (
	nodeTableName             = "pubsub_nodes"
	nodeAffiliationTableName  = "pubsub_affiliations"
	nodeSubscriptionTableName = "pubsub_subscriptions"
	nodeItemTableName         = "pubsub_items"
)

type pgSQLPubSubRep struct {
	conn   conn
	logger kitlog.Logger
}

func (r *pgSQLPubSubRep) UpsertNode(ctx context.Context, node *pubsubmodel.Node) error {
	optBytes, err := proto.Marshal(node.Options)
	if err != nil {
		return err
	}
	_, err = sq.Insert(nodeTableName).
		Prefix(noLoadBalancePrefix).
		Columns("host", "name", "options").
		Values(node.Host, node.Name, optBytes).
		Suffix("ON CONFLICT (host, name) DO UPDATE SET options = $3").
		RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
	q := sq.Select("id", "host", "name", "options").
		From(nodeTableName).
		Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}})

	return scanNode(q.RunWith(r.conn).QueryRowContext(ctx))
}

func (r *pgSQLPubSubRep) FetchNodes(ctx context.Context, host string) ([]*pubsubmodel.Node, error) {
	q := sq.Select("id", "host", "name", "options").
		From(nodeTableName).
		Where(sq.Eq{"host": host})

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	return scanNodes(rows)
}

func (r *pgSQLPubSubRep) NodeExists(ctx context.Context, host, name string) (bool, error) {
	q := sq.Select("COUNT(*)").
		From(nodeTableName).
		Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}})

	var count int
	err := q.RunWith(r.conn).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}

func (r *pgSQLPubSubRep) DeleteNode(ctx context.Context, host, name string) error {
	_, err := sq.Delete(nodeTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) DeleteNodes(ctx context.Context, host string) error {
	_, err := sq.Delete(nodeTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Eq{"host": host}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
	_, err := sq.Insert(nodeAffiliationTableName).
		Prefix(noLoadBalancePrefix).
		Columns("node_id", "jid", "affiliation").
		Values(sq.Expr("(SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name), affiliation.Jid, pubsubmodel.Affiliation2String[affiliation.State]).
		Suffix("ON CONFLICT (node_id, jid) DO UPDATE SET affiliation = $4").
		RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) FetchNodeAffiliation(ctx context.Context, jid, host, name string) (*pubsubmodel.Affiliation, error) {
	q := sq.Select("node_id", "jid", "affiliation").
		From(nodeAffiliationTableName).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name)).
		Where(sq.Eq{"jid": jid})

	return scanNodeAffiliation(q.RunWith(r.conn).QueryRowContext(ctx))
}

func (r *pgSQLPubSubRep) FetchNodeAffiliations(ctx context.Context, host, name string) ([]*pubsubmodel.Affiliation, error) {
	q := sq.Select("node_id", "jid", "affiliation").
		From(nodeAffiliationTableName).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name))

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	return scanNodeAffiliations(rows)
}

func (r *pgSQLPubSubRep) DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error {
	_, err := sq.Delete(nodeAffiliationTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name)).
		Where(sq.Eq{"jid": jid}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) DeleteNodeAffiliations(ctx context.Context, host, name string) error {
	_, err := sq.Delete(nodeAffiliationTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name)).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
	_, err := sq.Insert(nodeSubscriptionTableName).
		Prefix(noLoadBalancePrefix).
		Columns("node_id", "id", "jid", "subscription").
		Values(sq.Expr("(SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name), subscription.Id, subscription.Jid, pubsubmodel.Subscription2String[subscription.State]).
		Suffix("ON CONFLICT (node_id, jid) DO UPDATE SET subscription = $5").
		RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) FetchNodeSubscription(ctx context.Context, jid, host, name string) (*pubsubmodel.Subscription, error) {
	q := sq.Select("node_id", "id", "jid", "subscription").
		From(nodeSubscriptionTableName).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name)).
		Where(sq.Eq{"jid": jid})

	return scanNodeSubscription(q.RunWith(r.conn).QueryRowContext(ctx))
}

func (r *pgSQLPubSubRep) FetchNodeSubscriptions(ctx context.Context, host, name string) ([]*pubsubmodel.Subscription, error) {
	q := sq.Select("node_id", "id", "jid", "subscription").
		From(nodeSubscriptionTableName).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name))

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	return scanNodeSubscriptions(rows)
}

func (r *pgSQLPubSubRep) DeleteNodeSubscription(ctx context.Context, jid, host, name string) error {
	_, err := sq.Delete(nodeSubscriptionTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name)).
		Where(sq.Eq{"jid": jid}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) DeleteNodeSubscriptions(ctx context.Context, host, name string) error {
	_, err := sq.Delete(nodeSubscriptionTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name)).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) InsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string) error {
	payload, err := proto.Marshal(item.Payload)
	if err != nil {
		return err
	}
	q := sq.Insert(nodeItemTableName).
		Prefix(noLoadBalancePrefix).
		Columns("node_id", "id", "publisher", "payload").
		Values(sq.Expr("(SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name), item.Id, item.Publisher, payload)

	_, err = q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) FetchNodeItems(ctx context.Context, host, name string) ([]*pubsubmodel.Item, error) {
	q := sq.Select("node_id", "id", "publisher", "payload").
		From(nodeItemTableName).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name))

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	return scanNodeItems(rows)
}

func (r *pgSQLPubSubRep) DeleteOldestNodeItems(ctx context.Context, host, name string, maxItems int) error {
	q := sq.Delete(nodeItemTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.And{
			sq.Expr(`node_id = (SELECT "id" FROM pubsub_nodes WHERE host = ? AND name = ?)`, host, name),
			sq.Expr(`"id" NOT IN (SELECT "id" FROM pubsub_items WHERE host = ? AND name = ? ORDER BY created_at DESC LIMIT ? OFFSET 0)`, host, name, maxItems),
		})
	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLPubSubRep) DeleteNodeItems(ctx context.Context, host, name string) error {
	_, err := sq.Delete(nodeItemTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name)).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func scanNodes(scanner rowsScanner) ([]*pubsubmodel.Node, error) {
	var nodes []*pubsubmodel.Node

	for scanner.Next() {
		node, err := scanNode(scanner)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func scanNode(scanner rowScanner) (*pubsubmodel.Node, error) {
	var node pubsubmodel.Node
	var optBytes []byte

	err := scanner.Scan(&node.Id, &node.Host, &node.Name, &optBytes)
	switch err {
	case nil:
		var opts pubsubmodel.Options
		if err := proto.Unmarshal(optBytes, &opts); err != nil {
			return nil, err
		}
		node.Options = &opts
		return &node, nil

	case sql.ErrNoRows:
		return nil, nil

	default:
		return nil, err
	}
}

func scanNodeAffiliations(scanner rowsScanner) ([]*pubsubmodel.Affiliation, error) {
	var affs []*pubsubmodel.Affiliation

	for scanner.Next() {
		aff, err := scanNodeAffiliation(scanner)
		if err != nil {
			return nil, err
		}
		affs = append(affs, aff)
	}
	return affs, nil
}

func scanNodeAffiliation(scanner rowScanner) (*pubsubmodel.Affiliation, error) {
	var aff pubsubmodel.Affiliation
	var state string

	err := scanner.Scan(&aff.NodeId, &aff.Jid, &state)
	switch err {
	case nil:
		aff.State = pubsubmodel.String2Affiliation[state]
		return &aff, nil

	case sql.ErrNoRows:
		return nil, nil

	default:
		return nil, err
	}
}

func scanNodeSubscriptions(scanner rowsScanner) ([]*pubsubmodel.Subscription, error) {
	var subs []*pubsubmodel.Subscription

	for scanner.Next() {
		sub, err := scanNodeSubscription(scanner)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func scanNodeSubscription(scanner rowScanner) (*pubsubmodel.Subscription, error) {
	var sub pubsubmodel.Subscription
	var state string

	err := scanner.Scan(&sub.NodeId, &sub.Id, &sub.Jid, &state)
	switch err {
	case nil:
		sub.State = pubsubmodel.String2Subscription[state]
		return &sub, nil

	case sql.ErrNoRows:
		return nil, nil

	default:
		return nil, err
	}
}

func scanNodeItems(scanner rowsScanner) ([]*pubsubmodel.Item, error) {
	var items []*pubsubmodel.Item

	for scanner.Next() {
		sub, err := scanNodeItem(scanner)
		if err != nil {
			return nil, err
		}
		items = append(items, sub)
	}
	return items, nil
}

func scanNodeItem(scanner rowScanner) (*pubsubmodel.Item, error) {
	var itm pubsubmodel.Item
	var payload []byte

	err := scanner.Scan(&itm.NodeId, &itm.Id, &itm.Publisher, &payload)
	switch err {
	case nil:
		var msg stravaganza.PBElement
		if err := proto.Unmarshal(payload, &msg); err != nil {
			return nil, err
		}
		itm.Payload = &msg
		return &itm, nil

	case sql.ErrNoRows:
		return nil, nil

	default:
		return nil, err
	}
}
