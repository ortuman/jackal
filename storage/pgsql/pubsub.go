/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/xmpp"
)

func (s *Storage) FetchHosts(ctx context.Context) ([]string, error) {
	rows, err := sq.Select("DISTINCT(host)").
		From("pubsub_nodes").
		RunWith(s.db).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var hosts []string
	for rows.Next() {
		var host string
		if err := rows.Scan(&host); err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

func (s *Storage) UpsertNode(ctx context.Context, node *pubsubmodel.Node) error {
	return s.inTransaction(ctx, func(tx *sql.Tx) error {
		// if not existing, insert new node
		_, err := sq.Insert("pubsub_nodes").
			Columns("host", "name", "updated_at", "created_at").
			Suffix("ON CONFLICT (host, name) DO NOTHING").
			Values(node.Host, node.Name, nowExpr, nowExpr).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}

		// fetch node identifier
		var nodeIdentifier string

		err = sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": node.Host}, sq.Eq{"name": node.Name}}).
			RunWith(tx).QueryRowContext(ctx).Scan(&nodeIdentifier)
		if err != nil {
			return err
		}

		// delete previous node options
		_, err = sq.Delete("pubsub_node_options").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// insert new option set
		optionSetMap, err := node.Options.Map()
		if err != nil {
			return err
		}
		for name, value := range optionSetMap {
			_, err = sq.Insert("pubsub_node_options").
				Columns("node_id", "name", "value").
				Values(nodeIdentifier, name, value).
				RunWith(tx).ExecContext(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
	opts, err := s.fetchPubSubNodeOptions(ctx, host, name)
	if err != nil {
		return nil, err
	}
	return &pubsubmodel.Node{
		Host:    host,
		Name:    name,
		Options: *opts,
	}, nil
}

func (s *Storage) FetchNodes(ctx context.Context, host string) ([]pubsubmodel.Node, error) {
	rows, err := sq.Select("name").
		From("pubsub_nodes").
		Where(sq.Eq{"host": host}).
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var nodes []pubsubmodel.Node
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		var node = pubsubmodel.Node{Host: host, Name: name}
		opts, err := s.fetchPubSubNodeOptions(ctx, host, name)
		if err != nil {
			return nil, err
		}
		if opts != nil {
			node.Options = *opts
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *Storage) FetchSubscribedNodes(ctx context.Context, jid string) ([]pubsubmodel.Node, error) {
	rows, err := sq.Select("host", "name").
		From("pubsub_nodes").
		Where(sq.Expr("id IN (SELECT DISTINCT(node_id) FROM pubsub_subscriptions WHERE jid = $1 AND subscription = $2)", jid, pubsubmodel.Subscribed)).
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var nodes []pubsubmodel.Node
	for rows.Next() {
		var host, name string
		if err := rows.Scan(&host, &name); err != nil {
			return nil, err
		}
		var node = pubsubmodel.Node{Host: host, Name: name}
		opts, err := s.fetchPubSubNodeOptions(ctx, host, name)
		if err != nil {
			return nil, err
		}
		if opts != nil {
			node.Options = *opts
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *Storage) DeleteNode(ctx context.Context, host, name string) error {
	return s.inTransaction(ctx, func(tx *sql.Tx) error {
		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRowContext(ctx).Scan(&nodeIdentifier)
		switch err {
		case nil:
			break
		case sql.ErrNoRows:
			return nil
		default:
			return err
		}
		// delete node
		_, err = sq.Delete("pubsub_nodes").
			Where(sq.Eq{"id": nodeIdentifier}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// delete options
		_, err = sq.Delete("pubsub_node_options").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// delete items
		_, err = sq.Delete("pubsub_items").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// delete affiliations
		_, err = sq.Delete("pubsub_affiliations").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// delete subscriptions
		_, err = sq.Delete("pubsub_subscriptions").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).ExecContext(ctx)
		return err
	})
}

func (s *Storage) UpsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return s.inTransaction(ctx, func(tx *sql.Tx) error {
		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRowContext(ctx).Scan(&nodeIdentifier)
		switch err {
		case nil:
			break
		case sql.ErrNoRows:
			return nil
		default:
			return err
		}

		// upsert new item
		rawPayload := item.Payload.String()

		_, err = sq.Insert("pubsub_items").
			Columns("node_id", "item_id", "payload", "publisher").
			Values(nodeIdentifier, item.ID, rawPayload, item.Publisher).
			Suffix("ON CONFLICT (node_id, item_id) DO UPDATE SET payload = $5, publisher = $6", rawPayload, item.Publisher).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}

		// check if maximum item count was reached and delete oldest one
		_, err = sq.Delete("pubsub_items").
			Where("item_id IN (SELECT item_id FROM pubsub_items WHERE node_id = $1 ORDER BY created_at DESC OFFSET $2)", nodeIdentifier, maxNodeItems).
			RunWith(tx).ExecContext(ctx)
		return err
	})
}

func (s *Storage) FetchNodeItems(ctx context.Context, host, name string) ([]pubsubmodel.Item, error) {
	rows, err := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name).
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanPubSubNodeItems(rows)
}

func (s *Storage) FetchNodeItemsWithIDs(ctx context.Context, host, name string, identifiers []string) ([]pubsubmodel.Item, error) {
	rows, err := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where(sq.And{sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name), sq.Eq{"id": identifiers}}).
		OrderBy("created_at").
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanPubSubNodeItems(rows)
}

func (s *Storage) FetchNodeLastItem(ctx context.Context, host, name string) (*pubsubmodel.Item, error) {
	row := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name).
		OrderBy("created_at DESC").
		Limit(1).
		RunWith(s.db).QueryRowContext(ctx)

	item, err := s.scanPubSubNodeItem(row)
	switch err {
	case nil:
		return item, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *Storage) UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
	return s.inTransaction(ctx, func(tx *sql.Tx) error {
		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRowContext(ctx).Scan(&nodeIdentifier)
		switch err {
		case nil:
			break
		case sql.ErrNoRows:
			return nil
		default:
			return err
		}

		// upsert affiliation
		_, err = sq.Insert("pubsub_affiliations").
			Columns("node_id", "jid", "affiliation").
			Values(nodeIdentifier, affiliation.JID, affiliation.Affiliation).
			Suffix("ON CONFLICT (node_id, jid) DO UPDATE SET affiliation = $4", affiliation.Affiliation).
			RunWith(tx).ExecContext(ctx)
		return err
	})
}

func (s *Storage) FetchNodeAffiliation(ctx context.Context, host, name, jid string) (*pubsubmodel.Affiliation, error) {
	var aff pubsubmodel.Affiliation

	row := sq.Select("jid", "affiliation").
		From("pubsub_affiliations").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2) AND jid = $3", host, name, jid).
		RunWith(s.db).QueryRowContext(ctx)
	err := row.Scan(&aff.JID, &aff.Affiliation)
	switch err {
	case nil:
		return &aff, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *Storage) FetchNodeAffiliations(ctx context.Context, host, name string) ([]pubsubmodel.Affiliation, error) {
	rows, err := sq.Select("jid", "affiliation").
		From("pubsub_affiliations").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name).
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanPubSubNodeAffiliations(rows)
}

func (s *Storage) DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error {
	_, err := sq.Delete("pubsub_affiliations").
		Where("jid = $1 AND node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", jid, host, name).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *Storage) UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
	return s.inTransaction(ctx, func(tx *sql.Tx) error {
		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRowContext(ctx).Scan(&nodeIdentifier)
		switch err {
		case nil:
			break
		case sql.ErrNoRows:
			return nil
		default:
			return err
		}

		// upsert subscription
		_, err = sq.Insert("pubsub_subscriptions").
			Columns("node_id", "subid", "jid", "subscription", "updated_at", "created_at").
			Values(nodeIdentifier, subscription.SubID, subscription.JID, subscription.Subscription, nowExpr, nowExpr).
			Suffix("ON CONFLICT (node_id, jid) DO UPDATE SET subid = $5, subscription = $6", subscription.SubID, subscription.Subscription).
			RunWith(tx).ExecContext(ctx)
		return err
	})
}

func (s *Storage) FetchNodeSubscriptions(ctx context.Context, host, name string) ([]pubsubmodel.Subscription, error) {
	rows, err := sq.Select("subid", "jid", "subscription").
		From("pubsub_subscriptions").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name).
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanPubSubNodeSubscriptions(rows)
}

func (s *Storage) DeleteNodeSubscription(ctx context.Context, jid, host, name string) error {
	_, err := sq.Delete("pubsub_subscriptions").
		Where("jid = $1 AND node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", jid, host, name).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *Storage) fetchPubSubNodeOptions(ctx context.Context, host, name string) (*pubsubmodel.Options, error) {
	rows, err := sq.Select("name", "value").
		From("pubsub_node_options").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name).
		OrderBy("created_at").
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var optMap = make(map[string]string)
	for rows.Next() {
		var opt, value string
		if err := rows.Scan(&opt, &value); err != nil {
			return nil, err
		}
		optMap[opt] = value
	}
	if len(optMap) == 0 {
		return nil, nil // node does not exist
	}
	opts, err := pubsubmodel.NewOptionsFromMap(optMap)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func (s *Storage) scanPubSubNodeAffiliations(scanner rowsScanner) ([]pubsubmodel.Affiliation, error) {
	var affiliations []pubsubmodel.Affiliation

	for scanner.Next() {
		var affiliation pubsubmodel.Affiliation
		if err := scanner.Scan(&affiliation.JID, &affiliation.Affiliation); err != nil {
			return nil, err
		}
		affiliations = append(affiliations, affiliation)
	}
	return affiliations, nil
}

func (s *Storage) scanPubSubNodeSubscriptions(scanner rowsScanner) ([]pubsubmodel.Subscription, error) {
	var subscriptions []pubsubmodel.Subscription

	for scanner.Next() {
		var subscription pubsubmodel.Subscription
		if err := scanner.Scan(&subscription.SubID, &subscription.JID, &subscription.Subscription); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, nil
}

func (s *Storage) scanPubSubNodeItems(scanner rowsScanner) ([]pubsubmodel.Item, error) {
	var items []pubsubmodel.Item
	var err error

	for scanner.Next() {
		var payload string
		var item pubsubmodel.Item
		if err := scanner.Scan(&item.ID, &item.Publisher, &payload); err != nil {
			return nil, err
		}
		parser := xmpp.NewParser(strings.NewReader(payload), xmpp.DefaultMode, 0)
		item.Payload, err = parser.ParseElement()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Storage) scanPubSubNodeItem(scanner rowScanner) (*pubsubmodel.Item, error) {
	var payload string
	var item pubsubmodel.Item
	var err error

	if err = scanner.Scan(&item.ID, &item.Publisher, &payload); err != nil {
		return nil, err
	}
	parser := xmpp.NewParser(strings.NewReader(payload), xmpp.DefaultMode, 0)
	item.Payload, err = parser.ParseElement()
	if err != nil {
		return nil, err
	}
	return &item, nil
}
