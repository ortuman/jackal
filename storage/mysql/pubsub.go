package mysql

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/xmpp"
)

func (s *Storage) UpsertPubSubNode(node *pubsubmodel.Node) error {
	return s.inTransaction(func(tx *sql.Tx) error {

		// if not existing, insert new node
		_, err := sq.Insert("pubsub_nodes").
			Columns("host", "name", "updated_at", "created_at").
			Suffix("ON DUPLICATE KEY UPDATE updated_at = NOW()").
			Values(node.Host, node.Name, nowExpr, nowExpr).
			RunWith(tx).Exec()
		if err != nil {
			return err
		}

		// fetch node identifier
		var nodeIdentifier string

		err = sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": node.Host}, sq.Eq{"name": node.Name}}).
			RunWith(tx).QueryRow().Scan(&nodeIdentifier)
		if err != nil {
			return err
		}
		// delete previous node options
		_, err = sq.Delete("pubsub_node_options").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).Exec()
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
				RunWith(tx).Exec()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	rows, err := sq.Select("name", "value").
		From("pubsub_node_options").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		RunWith(s.db).Query()
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
	opts, err := pubsubmodel.NewOptionsFromMap(optMap)
	if err != nil {
		return nil, err
	}
	return &pubsubmodel.Node{
		Host:    host,
		Name:    name,
		Options: *opts,
	}, nil
}

func (s *Storage) DeletePubSubNode(host, name string) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRow().Scan(&nodeIdentifier)
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
			RunWith(tx).Exec()
		if err != nil {
			return err
		}
		// delete options
		_, err = sq.Delete("pubsub_node_options").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).Exec()
		if err != nil {
			return err
		}
		// delete items
		_, err = sq.Delete("pubsub_items").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).Exec()
		if err != nil {
			return err
		}
		// delete affiliations
		_, err = sq.Delete("pubsub_affiliations").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).Exec()
		if err != nil {
			return err
		}
		// delete subscriptions
		_, err = sq.Delete("pubsub_subscriptions").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).Exec()
		return err
	})
}

func (s *Storage) UpsertPubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRow().Scan(&nodeIdentifier)
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
			RunWith(tx).Exec()

		// get total items count
		var itemsCount int

		err = sq.Select("COUNT(*)").
			From("pubsub_items").
			Where(sq.Eq{"node_id": nodeIdentifier}).
			RunWith(tx).QueryRow().Scan(&itemsCount)
		if err != nil {
			return err
		}

		// check if maximum item count was reached
		if itemsCount == maxNodeItems {
			// fetch oldest item timestamp
			var oldestCreatedAt string

			err := sq.Select("MIN(created_at)").
				From("pubsub_items").
				Where(sq.Eq{"node_id": nodeIdentifier}).
				RunWith(tx).QueryRow().Scan(&oldestCreatedAt)
			if err != nil {
				return err
			}
			// delete oldest item
			_, err = sq.Delete("pubsub_items").
				Where(sq.And{sq.Eq{"node_id": nodeIdentifier}, sq.Eq{"created_at": oldestCreatedAt}}).
				RunWith(tx).Exec()
			if err != nil {
				return err
			}
		}
		return err
	})
}

func (s *Storage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	rows, err := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		OrderBy("created_at").
		RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []pubsubmodel.Item
	for rows.Next() {
		var payload string
		var item pubsubmodel.Item
		if err := rows.Scan(&item.ID, &item.Publisher, &payload); err != nil {
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

func (s *Storage) FetchPubSubNodeItemsWithIDs(host, name string, identifiers []string) ([]pubsubmodel.Item, error) {
	// TODO(ortuman): implement me!
	return nil, nil
}

func (s *Storage) UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return s.inTransaction(func(tx *sql.Tx) error {

		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRow().Scan(&nodeIdentifier)
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
			RunWith(tx).Exec()
		return err
	})
}

func (s *Storage) FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	rows, err := sq.Select("jid", "affiliation").
		From("pubsub_affiliations").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var affiliations []pubsubmodel.Affiliation
	for rows.Next() {
		var affiliation pubsubmodel.Affiliation
		if err := rows.Scan(&affiliation.JID, &affiliation.Affiliation); err != nil {
			return nil, err
		}
		affiliations = append(affiliations, affiliation)
	}
	return affiliations, nil
}

func (s *Storage) DeletePubSubNodeAffiliation(jid, host, name string) error {
	_, err := sq.Delete("pubsub_affiliations").
		Where("jid = ? AND node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", jid, host, name).
		RunWith(s.db).Exec()
	return err
}

func (s *Storage) UpsertPubSubNodeSubscription(subscription *pubsubmodel.Subscription, host, name string) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		// fetch node identifier
		var nodeIdentifier string

		err := sq.Select("id").
			From("pubsub_nodes").
			Where(sq.And{sq.Eq{"host": host}, sq.Eq{"name": name}}).
			RunWith(tx).QueryRow().Scan(&nodeIdentifier)
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
			Columns("node_id", "subid", "jid", "subscription").
			Values(nodeIdentifier, subscription.SubID, subscription.JID, subscription.Subscription).
			Suffix("ON DUPLICATE KEY UPDATE subid = ?, subscription = ?, updated_at = NOW()", subscription.SubID, subscription.Subscription).
			RunWith(tx).Exec()
		return err
	})
}

func (s *Storage) FetchPubSubNodeSubscriptions(host, name string) ([]pubsubmodel.Subscription, error) {
	rows, err := sq.Select("subid", "jid", "subscription").
		From("pubsub_subscriptions").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", host, name).
		RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var subscriptions []pubsubmodel.Subscription
	for rows.Next() {
		var subscription pubsubmodel.Subscription
		if err := rows.Scan(&subscription.SubID, &subscription.JID, &subscription.Subscription); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, nil
}

func (s *Storage) DeletePubSubNodeSubscription(jid, host, name string) error {
	_, err := sq.Delete("pubsub_subscriptions").
		Where("jid = ? AND node_id = (SELECT id FROM pubsub_nodes WHERE host = ? AND name = ?)", jid, host, name).
		RunWith(s.db).Exec()
	return err
}
