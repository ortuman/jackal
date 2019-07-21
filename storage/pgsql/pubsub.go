package pgsql

import (
	"database/sql"
	"errors"
	"strings"

	sq "github.com/Masterminds/squirrel"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/xmpp"
)

func (m *Storage) InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error {
	return errors.New("unimplemented method")
}

func (s *Storage) InsertOrUpdatePubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
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
			Columns("node_id", "item_id", "payload", "publisher").
			Values(nodeIdentifier, item.ID, rawPayload, item.Publisher).
			Suffix("ON CONFLICT (node_id, item_id) DO UPDATE SET payload = $5, publisher = $6", rawPayload, item.Publisher).
			RunWith(s.db).Exec()
		if err != nil {
			return err
		}

		// check if maximum item count was reached and delete oldest one
		_, err = sq.Delete("pubsub_items").
			Where("item_id IN (SELECT item_id FROM pubsub_items WHERE node_id = $1 ORDER BY created_at DESC OFFSET $2)", nodeIdentifier, maxNodeItems).
			RunWith(tx).Exec()
		return err
	})
}

func (s *Storage) GetPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	rows, err := sq.Select("item_id", "publisher", "payload").
		From("pubsub_items").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name).
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

func (s *Storage) InsertOrUpdatePubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
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

		// upsert affiliation
		_, err = sq.Insert("pubsub_affiliations").
			Columns("node_id", "jid", "affiliation").
			Values(nodeIdentifier, affiliation.JID, affiliation.Affiliation).
			Suffix("ON CONFLICT (node_id, jid) DO UPDATE SET affiliation = $4", affiliation.Affiliation).
			RunWith(s.db).Exec()
		return err
	})
}

func (s *Storage) GetPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	rows, err := sq.Select("jid", "affiliation").
		From("pubsub_affiliations").
		Where("node_id = (SELECT id FROM pubsub_nodes WHERE host = $1 AND name = $2)", host, name).
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
