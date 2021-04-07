// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgsqlrepository

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	blocklistmodel "github.com/ortuman/jackal/model/blocklist"
)

const (
	blockListsTableName = "blocklist_items"
)

type pgSQLBlockListRep struct {
	conn conn
}

func (r *pgSQLBlockListRep) UpsertBlockListItem(ctx context.Context, item *blocklistmodel.Item) error {
	_, err := sq.Insert(blockListsTableName).
		Columns("username", "jid").
		Values(item.Username, item.JID).
		Suffix("ON CONFLICT (username, jid) DO NOTHING").
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLBlockListRep) DeleteBlockListItem(ctx context.Context, item *blocklistmodel.Item) error {
	_, err := sq.Delete(blockListsTableName).
		Where(sq.And{sq.Eq{"username": item.Username}, sq.Eq{"jid": item.JID}}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLBlockListRep) FetchBlockListItems(ctx context.Context, username string) ([]blocklistmodel.Item, error) {
	q := sq.Select("username", "jid").
		From(blockListsTableName).
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	return scanBlockListItems(rows)
}

func scanBlockListItems(scanner rowsScanner) ([]blocklistmodel.Item, error) {
	var ret []blocklistmodel.Item
	for scanner.Next() {
		var it blocklistmodel.Item
		if err := scanner.Scan(&it.Username, &it.JID); err != nil {
			return nil, err
		}
		ret = append(ret, it)
	}
	return ret, nil
}
