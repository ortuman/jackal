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
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/lib/pq"
	rostermodel "github.com/ortuman/jackal/model/roster"
)

const (
	rosterVersionsTableName      = "roster_versions"
	rosterItemsTableName         = "roster_items"
	rosterNotificationsTableName = "roster_notifications"
)

type pgSQLRosterRep struct {
	conn conn
}

func (r *pgSQLRosterRep) TouchRosterVersion(ctx context.Context, username string) (int, error) {
	b := sq.Insert(rosterVersionsTableName).
		Columns("username").
		Values(username).
		Suffix("ON CONFLICT (username) DO UPDATE SET ver = roster_versions.ver + 1").
		Suffix("RETURNING ver")

	var ver int
	err := b.RunWith(r.conn).QueryRowContext(ctx).Scan(&ver)
	if err != nil {
		return 0, err
	}
	return ver, nil
}

func (r *pgSQLRosterRep) FetchRosterVersion(ctx context.Context, username string) (int, error) {
	q := sq.Select("ver").
		From(rosterVersionsTableName).
		Where(sq.Eq{"username": username})

	var ver int
	err := q.RunWith(r.conn).QueryRowContext(ctx).Scan(&ver)
	switch err {
	case nil:
		return ver, nil
	case sql.ErrNoRows:
		return 0, nil
	default:
		return 0, err
	}
}

func (r *pgSQLRosterRep) UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) error {
	q := sq.Insert(rosterItemsTableName).
		Columns("username", "jid", "name", "subscription", "groups", "ask").
		Values(ri.Username, ri.JID, ri.Name, ri.Subscription, pq.Array(ri.Groups), ri.Ask).
		Suffix("ON CONFLICT (username, jid) DO UPDATE SET name = $3, subscription = $4, groups = $5, ask = $6")

	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLRosterRep) DeleteRosterItem(ctx context.Context, username, jid string) error {
	_, err := sq.Delete(rosterItemsTableName).
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}}).
		RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLRosterRep) DeleteRosterItems(ctx context.Context, username string) error {
	_, err := sq.Delete(rosterItemsTableName).
		Where(sq.Eq{"username": username}).
		RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLRosterRep) FetchRosterItems(ctx context.Context, username string) ([]rostermodel.Item, error) {
	q := sq.Select("username", "jid", "name", "subscription", "groups", "ask").
		From(rosterItemsTableName).
		Where(sq.Eq{"username": username}).
		OrderBy("created_at DESC")

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	return scanRosterItems(rows)
}

func (r *pgSQLRosterRep) FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]rostermodel.Item, error) {
	q := sq.Select("username", "jid", "name", "subscription", "groups", "ask").
		From(rosterItemsTableName).
		Where(sq.Expr("username = $1 AND groups @> $2", username, pq.Array(groups))).
		OrderBy("created_at DESC")

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	return scanRosterItems(rows)
}

func (r *pgSQLRosterRep) FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error) {
	q := sq.Select("username", "jid", "name", "subscription", "groups", "ask").
		From(rosterItemsTableName).
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}})

	ri, err := scanRosterItem(q.RunWith(r.conn).QueryRowContext(ctx))
	switch err {
	case nil:
		return ri, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (r *pgSQLRosterRep) UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error {
	prBytes, err := rn.Presence.MarshalBinary()
	if err != nil {
		return err
	}
	q := sq.Insert(rosterNotificationsTableName).
		Columns("contact", "jid", "presence").
		Values(rn.Contact, rn.JID, prBytes).
		Suffix("ON CONFLICT (contact, jid) DO UPDATE SET presence = $3")

	_, err = q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLRosterRep) DeleteRosterNotification(ctx context.Context, contact, jid string) error {
	q := sq.Delete(rosterNotificationsTableName).
		Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})
	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLRosterRep) DeleteRosterNotifications(ctx context.Context, contact string) error {
	q := sq.Delete(rosterNotificationsTableName).
		Where(sq.Eq{"contact": contact})
	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLRosterRep) FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	q := sq.Select("contact", "jid", "presence").
		From(rosterNotificationsTableName).
		Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})

	rn, err := scanRosterNotification(q.RunWith(r.conn).QueryRowContext(ctx))
	switch err {
	case nil:
		return rn, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (r *pgSQLRosterRep) FetchRosterNotifications(ctx context.Context, contact string) ([]rostermodel.Notification, error) {
	q := sq.Select("contact", "jid", "presence").
		From(rosterNotificationsTableName).
		Where(sq.Eq{"contact": contact})

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	return scanRosterNotifications(rows)
}

func (r *pgSQLRosterRep) FetchRosterGroups(ctx context.Context, username string) ([]string, error) {
	q := sq.Select("DISTINCT UNNEST(groups)").
		From(rosterItemsTableName).
		Where(sq.Eq{"username": username})

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var groups []string
	for rows.Next() {
		var group string
		if err := rows.Scan(&group); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func scanRosterItem(scanner rowScanner) (*rostermodel.Item, error) {
	var ri rostermodel.Item
	err := scanner.Scan(
		&ri.Username,
		&ri.JID,
		&ri.Name,
		&ri.Subscription,
		pq.Array(&ri.Groups),
		&ri.Ask,
	)
	if err != nil {
		return nil, err
	}
	return &ri, nil
}

func scanRosterItems(scanner rowsScanner) ([]rostermodel.Item, error) {
	var ret []rostermodel.Item
	for scanner.Next() {
		ri, err := scanRosterItem(scanner)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *ri)
	}
	return ret, nil
}

func scanRosterNotification(scanner rowScanner) (*rostermodel.Notification, error) {
	var rn rostermodel.Notification

	var prBytes []byte
	if err := scanner.Scan(&rn.Contact, &rn.JID, &prBytes); err != nil {
		return nil, err
	}
	b, err := stravaganza.NewBuilderFromBinary(prBytes)
	if err != nil {
		return nil, err
	}
	pr, err := b.BuildPresence()
	if err != nil {
		return nil, err
	}
	rn.Presence = pr
	return &rn, nil
}

func scanRosterNotifications(scanner rowsScanner) ([]rostermodel.Notification, error) {
	var ret []rostermodel.Notification
	for scanner.Next() {
		ri, err := scanRosterNotification(scanner)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *ri)
	}
	return ret, nil
}
