/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/storage/model"
)

func (s *Storage) InsertOrUpdateBlockListItems(items []model.BlockListItem) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		for _, item := range items {
			_, err := sq.Insert("blocklist_items").
				Options("IGNORE").
				Columns("username", "jid", "created_at").
				Values(item.Username, item.JID, nowExpr).
				RunWith(tx).Exec()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) DeleteBlockListItems(items []model.BlockListItem) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		for _, item := range items {
			_, err := sq.Delete("blocklist_items").
				Where(sq.And{sq.Eq{"username": item.Username}, sq.Eq{"jid": item.JID}}).
				RunWith(tx).Exec()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	q := sq.Select("username", "jid").
		From("blocklist_items").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanBlockListItemEntities(rows)
}

func (s *Storage) scanBlockListItemEntities(scanner rowsScanner) ([]model.BlockListItem, error) {
	var ret []model.BlockListItem
	for scanner.Next() {
		var it model.BlockListItem
		scanner.Scan(&it.Username, &it.JID)
		ret = append(ret, it)
	}
	return ret, nil
}
