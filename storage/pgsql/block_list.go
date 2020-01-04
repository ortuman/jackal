/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
)

type pgSQLBlockList struct {
	*pgSQLStorage
}

func newBlockList(db *sql.DB) *pgSQLBlockList {
	return &pgSQLBlockList{
		pgSQLStorage: newStorage(db),
	}
}

func (s *pgSQLBlockList) InsertBlockListItem(ctx context.Context, item *model.BlockListItem) error {
	q := sq.Insert("blocklist_items").
		Columns("username", "jid").
		Values(item.Username, item.JID).
		RunWith(s.db)
	_, err := q.ExecContext(ctx)
	return err
}

func (s *pgSQLBlockList) DeleteBlockListItem(ctx context.Context, item *model.BlockListItem) error {
	q := sq.Delete("blocklist_items").
		Where(sq.And{sq.Eq{"username": item.Username}, sq.Eq{"jid": item.JID}}).
		RunWith(s.db)
	_, err := q.ExecContext(ctx)
	return err
}

func (s *pgSQLBlockList) FetchBlockListItems(ctx context.Context, username string) ([]model.BlockListItem, error) {
	q := sq.Select("username", "jid").
		From("blocklist_items").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanBlockListItemEntities(rows)
}

func scanBlockListItemEntities(scanner rowsScanner) ([]model.BlockListItem, error) {
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
