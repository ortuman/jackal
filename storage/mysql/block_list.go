/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
)

type mySQLBlockList struct {
	*mySQLStorage
}

func newBlockList(db *sql.DB) *mySQLBlockList {
	return &mySQLBlockList{
		mySQLStorage: newStorage(db),
	}
}

func (s *mySQLBlockList) InsertBlockListItem(ctx context.Context, item *model.BlockListItem) error {
	_, err := sq.Insert("blocklist_items").
		Options("IGNORE").
		Columns("username", "jid", "created_at").
		Values(item.Username, item.JID, nowExpr).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLBlockList) DeleteBlockListItem(ctx context.Context, item *model.BlockListItem) error {
	_, err := sq.Delete("blocklist_items").
		Where(sq.And{sq.Eq{"username": item.Username}, sq.Eq{"jid": item.JID}}).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLBlockList) FetchBlockListItems(ctx context.Context, username string) ([]model.BlockListItem, error) {
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
