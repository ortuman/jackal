/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
)

// InsertBlockListItems inserts a set of block list item entities
// into storage, only in case they haven't been previously inserted.
func (s *Storage) InsertBlockListItems(items []model.BlockListItem) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		for _, item := range items {
			q := sq.Insert("blocklist_items").
				Columns("username", "jid").
				Values(item.Username, item.JID).
				RunWith(tx)

			if _, err := q.Exec(); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBlockListItems deletes a set of block list item entities from storage.
func (s *Storage) DeleteBlockListItems(items []model.BlockListItem) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		for _, item := range items {
			q := sq.Delete("blocklist_items").
				Where(sq.And{sq.Eq{"username": item.Username}, sq.Eq{"jid": item.JID}}).
				RunWith(tx)

			if _, err := q.Exec(); err != nil {
				return err
			}
		}
		return nil
	})
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
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
