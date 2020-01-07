/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/xmpp"
)

type mySQLPubSub struct {
	*mySQLStorage
}

func newPubSub(db *sql.DB) *mySQLPubSub {
	return &mySQLPubSub{
		mySQLStorage: newStorage(db),
	}
}

func (s *mySQLPubSub) FetchHosts(ctx context.Context) ([]string, error) {
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

func (s *mySQLPubSub) UpsertNode(ctx context.Context, node *pubsubmodel.Node) error {
	return s.inTransaction(ctx, func(tx *sql.Tx) error {

		// if not existing, insert new node
		_, err := sq.Insert("pubsub_nodes").
			Columns("host", "name", "updated_at", "created_at").
			Suffix("ON DUPLICATE KEY UPDATE updated_at = NOW()").
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
				Columns("node_id", "name", "value", "updated_at", "created_at").
				Values(nodeIdentifier, name, value, nowExpr, nowExpr).
				RunWith(tx).ExecContext(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *mySQLPubSub) FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
	opts, err := s.fetchPubSubNodeOptions(ctx, host, name)
	if err != nil {
		return nil, err
	}
	if opts == nil {
		return nil, nil // not found
	}
	return &pubsubmodel.Node{
		Host:    host,
		Name:    name,
		Options: *opts,
	}, nil
}

func (s *mySQLPubSub) FetchNodes(ctx context.Context, host string) ([]pubsubmodel.Node, error) {
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

func (s *mySQLPubSub) FetchSubscribedNodes(ctx context.Context, jid string) ([]pubsubmodel.Node, error) {
	rows, err := sq.Select("host", "name").
		From("pubsub_nodes").
		Where(sq.Expr("id IN (SELECT DISTINCT(node_id) FROM pubsub_subscriptions WHERE jid = ? AND subscription = ?)", jid, pubsubmodel.Subscribed)).
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

func (s *mySQLPubSub) DeleteNode(ctx context.Context, host, name string) error {
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

func (s *mySQLPubSub) UpsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
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
			Columns("node_id", "item_id", "payload", "publisher", "updated_at", "created_at").
			Values(nodeIdentifier, item.ID, rawPayload, item.Publisher, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE payload = ?, publisher = ?, updated_at = NOW()", rawPayload, item.Publisher).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}

		// fetch valid identifiers
		rows, err := sq.Select("item_id").
			From("pubsub_items").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			OrderBy("created_at DESC").
			Limit(uint64(maxNodeItems)).RunWith(tx).QueryContext(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()

		var validIdentifiers []string
		for rows.Next() {
			var identifier string
			if err := rows.Scan(&identifier); err != nil {
				return err
			}
			validIdentifiers = append(validIdentifiers, identifier)
		}
		// delete older items
		_, err = sq.Delete("pubsub_items").
			Where(sq.And{sq.Eq{"node_id": nodeIdentifier}, sq.NotEq{"item_id": validIdentifiers}}).
			RunWith(tx).
			ExecContext(ctx)
		return err
	})
}

func (s *mySQLPubSub) FetchNodeItems(ctx context.Context, host, name string) ([]pubsubmodel.Item, error) {
	rows, err := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		OrderBy("created_at").
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanPubSubNodeItems(rows)
}

func (s *mySQLPubSub) FetchNodeItemsWithIDs(ctx context.Context, host, name string, identifiers []string) ([]pubsubmodel.Item, error) {
	rows, err := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where(sq.And{sq.Expr("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name), sq.Eq{"id": identifiers}}).
		OrderBy("created_at").
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanPubSubNodeItems(rows)
}

func (s *mySQLPubSub) FetchNodeLastItem(ctx context.Context, host, name string) (*pubsubmodel.Item, error) {
	row := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		OrderBy("created_at DESC").
		Limit(1).
		RunWith(s.db).QueryRowContext(ctx)

	item, err := scanPubSubNodeItem(row)
	switch err {
	case nil:
		return item, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQLPubSub) UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
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

		// insert affiliation
		_, err = sq.Insert("pubsub_affiliations").
			Columns("node_id", "jid", "affiliation", "updated_at", "created_at").
			Values(nodeIdentifier, affiliation.JID, affiliation.Affiliation, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE affiliation = ?, updated_at = NOW()", affiliation.Affiliation).
			RunWith(tx).ExecContext(ctx)
		return err
	})
}

func (s *mySQLPubSub) FetchNodeAffiliation(ctx context.Context, host, name, jid string) (*pubsubmodel.Affiliation, error) {
	var aff pubsubmodel.Affiliation

	row := sq.Select("jid", "affiliation").
		From("pubsub_affiliations").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?) AND jid = ?", host, name, jid).
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

func (s *mySQLPubSub) FetchNodeAffiliations(ctx context.Context, host, name string) ([]pubsubmodel.Affiliation, error) {
	rows, err := sq.Select("jid", "affiliation").
		From("pubsub_affiliations").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanPubSubNodeAffiliations(rows)
}

func (s *mySQLPubSub) DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error {
	_, err := sq.Delete("pubsub_affiliations").
		Where("jid = ? AND node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", jid, host, name).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLPubSub) UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
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
			Suffix("ON DUPLICATE KEY UPDATE subid = ?, subscription = ?, updated_at = NOW()", subscription.SubID, subscription.Subscription).
			RunWith(tx).ExecContext(ctx)
		return err
	})
}

func (s *mySQLPubSub) FetchNodeSubscriptions(ctx context.Context, host, name string) ([]pubsubmodel.Subscription, error) {
	rows, err := sq.Select("subid", "jid", "subscription").
		From("pubsub_subscriptions").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanPubSubNodeSubscriptions(rows)
}

func (s *mySQLPubSub) DeleteNodeSubscription(ctx context.Context, jid, host, name string) error {
	_, err := sq.Delete("pubsub_subscriptions").
		Where("jid = ? AND node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", jid, host, name).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLPubSub) fetchPubSubNodeOptions(ctx context.Context, host, name string) (*pubsubmodel.Options, error) {
	rows, err := sq.Select("name", "value").
		From("pubsub_node_options").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
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

func scanPubSubNodeAffiliations(scanner rowsScanner) ([]pubsubmodel.Affiliation, error) {
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

func scanPubSubNodeSubscriptions(scanner rowsScanner) ([]pubsubmodel.Subscription, error) {
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

func scanPubSubNodeItems(scanner rowsScanner) ([]pubsubmodel.Item, error) {
	var items []pubsubmodel.Item

	for scanner.Next() {
		item, err := scanPubSubNodeItem(scanner)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func scanPubSubNodeItem(scanner rowScanner) (*pubsubmodel.Item, error) {
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
