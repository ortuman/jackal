package mysql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

func (s *Storage) InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		// if not existing, insert new node
		res, err := sq.Insert("pubsub_nodes").
			Columns("host", "name", "updated_at", "created_at").
			Suffix("ON DUPLICATE KEY UPDATE updated_at = NOW()").
			Values(node.Host, node.Name, nowExpr, nowExpr).
			RunWith(tx).Exec()
		if err != nil {
			return err
		}
		nodeIdentifier, err := res.LastInsertId()
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
		for _, opt := range node.Options {
			_, err = sq.Insert("pubsub_node_options").
				Columns("node_id", "name", "value").
				Values(nodeIdentifier, opt.Name, opt.Value).
				RunWith(tx).Exec()
			if err != nil {
				return err
			}
		}
		return nil
	})
}
