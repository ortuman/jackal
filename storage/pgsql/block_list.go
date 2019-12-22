/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
)

// InsertBlockListItem inserts a block list item entity
// into storage, only in case they haven't been previously inserted.
func (s *Storage) InsertBlockListItem(item *model.BlockListItem) error {
	q := sq.Insert("blocklist_items").
		Columns("username", "jid").
		Values(item.Username, item.JID).
		RunWith(s.db)
	_, err := q.Exec()
	return err
}

// DeleteBlockListItem deletes a block list item entity from storage.
func (s *Storage) DeleteBlockListItem(item *model.BlockListItem) error {
	q := sq.Delete("blocklist_items").
		Where(sq.And{sq.Eq{"username": item.Username}, sq.Eq{"jid": item.JID}}).
		RunWith(s.db)
	_, err := q.Exec()
	return err
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
	defer func() { _ = rows.Close() }()

	return s.scanBlockListItemEntities(rows)
}

func (s *Storage) scanBlockListItemEntities(scanner rowsScanner) ([]model.BlockListItem, error) {
	var ret []model.BlockListItem

	for scanner.Next() {
		var it model.BlockListItem
		if err := scanner.Scan(&it.Username, &it.JID); err != nil {
			return nil, err
		}
		ret = append(ret, it)
	}
	return ret, nil
}
