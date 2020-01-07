/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	sq "github.com/Masterminds/squirrel"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type pgSQLRoster struct {
	*pgSQLStorage
	pool *pool.BufferPool
}

func newRoster(db *sql.DB) *pgSQLRoster {
	return &pgSQLRoster{
		pgSQLStorage: newStorage(db),
	}
}

func (s *pgSQLRoster) UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) (rostermodel.Version, error) {
	var ver rostermodel.Version

	err := s.inTransaction(ctx, func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username").
			Values(ri.Username).
			Suffix("ON CONFLICT (username) DO UPDATE SET ver = roster_versions.ver + 1")

		if _, err := q.RunWith(tx).ExecContext(ctx); err != nil {
			return err
		}
		groupsBytes, err := json.Marshal(ri.Groups)
		if err != nil {
			return err
		}

		verExpr := sq.Expr("(SELECT ver FROM roster_versions WHERE username = ?)", ri.Username)
		q = sq.Insert("roster_items").
			Columns("username", "jid", "name", "subscription", "groups", "ask", "ver").
			Values(ri.Username, ri.JID, ri.Name, ri.Subscription, groupsBytes, ri.Ask, verExpr).
			Suffix("ON CONFLICT (username, jid) DO UPDATE SET name = $3, subscription = $4, groups = $5, ask = $6, ver = roster_items.ver + 1")
		_, err = q.RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// delete previous groups
		_, err = sq.Delete("roster_groups").
			Where(sq.And{sq.Eq{"username": ri.Username}, sq.Eq{"jid": ri.JID}}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// insert groups
		for _, group := range ri.Groups {
			q = sq.Insert("roster_groups").
				Columns("username", "jid", `"group"`, "created_at", "updated_at").
				Values(ri.Username, ri.JID, group, nowExpr, nowExpr)
			_, err := q.RunWith(tx).ExecContext(ctx)
			if err != nil {
				return err
			}
		}
		// fetch new roster version
		ver, err = fetchRosterVer(ctx, ri.Username, tx)
		return err
	})
	if err != nil {
		return rostermodel.Version{}, err
	}
	return ver, nil
}

func (s *pgSQLRoster) DeleteRosterItem(ctx context.Context, username, jid string) (rostermodel.Version, error) {
	var ver rostermodel.Version

	err := s.inTransaction(ctx, func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username").
			Values(username).
			Suffix("ON CONFLICT (username) DO UPDATE SET ver = roster_versions.ver + 1, last_deletion_ver = roster_versions.ver")

		if _, err := q.RunWith(tx).ExecContext(ctx); err != nil {
			return err
		}
		// delete groups
		_, err := sq.Delete("roster_groups").
			Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		// delete items
		_, err = sq.Delete("roster_items").
			Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}

		// fetch new roster version
		ver, err = fetchRosterVer(ctx, username, tx)
		return err
	})
	if err != nil {
		return rostermodel.Version{}, err
	}
	return ver, nil
}

func (s *pgSQLRoster) FetchRosterItems(ctx context.Context, username string) ([]rostermodel.Item, rostermodel.Version, error) {
	q := sq.Select("username", "jid", "name", "subscription", "groups", "ask", "ver").
		From("roster_items").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at DESC")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	defer func() { _ = rows.Close() }()

	items, err := scanRosterItemEntities(rows)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	ver, err := fetchRosterVer(ctx, username, s.db)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	return items, ver, nil
}

func (s *pgSQLRoster) FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	q := sq.Select("ris.username", "ris.jid", "ris.name", "ris.subscription", "ris.groups", "ris.ask", "ris.ver").
		From("roster_items ris").
		LeftJoin("roster_groups g ON ris.username = g.username").
		Where(sq.And{sq.Eq{"ris.username": username}, sq.Eq{"g.group": groups}}).
		OrderBy("ris.created_at DESC")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	defer func() { _ = rows.Close() }()

	items, err := scanRosterItemEntities(rows)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	ver, err := fetchRosterVer(ctx, username, s.db)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	return items, ver, nil
}

func (s *pgSQLRoster) FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error) {
	q := sq.Select("username", "jid", "name", "subscription", "groups", "ask", "ver").
		From("roster_items").
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}})

	var ri rostermodel.Item
	err := scanRosterItemEntity(&ri, q.RunWith(s.db).QueryRowContext(ctx))
	switch err {
	case nil:
		return &ri, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *pgSQLRoster) UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error {
	presenceXML := rn.Presence.String()

	q := sq.Insert("roster_notifications").
		Columns("contact", "jid", "elements").
		Values(rn.Contact, rn.JID, presenceXML).
		Suffix("ON CONFLICT (contact, jid) DO UPDATE SET elements = $4", presenceXML)

	_, err := q.RunWith(s.db).ExecContext(ctx)

	return err
}

func (s *pgSQLRoster) FetchRosterNotifications(ctx context.Context, contact string) ([]rostermodel.Notification, error) {
	q := sq.Select("contact", "jid", "elements").
		From("roster_notifications").
		Where(sq.Eq{"contact": contact}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ret []rostermodel.Notification
	for rows.Next() {
		var rn rostermodel.Notification
		if err := scanRosterNotificationEntity(&rn, rows); err != nil {
			return nil, err
		}
		ret = append(ret, rn)
	}
	return ret, nil
}

func (s *pgSQLRoster) FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	q := sq.Select("contact", "jid", "elements").
		From("roster_notifications").
		Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})

	var rn rostermodel.Notification
	err := scanRosterNotificationEntity(&rn, q.RunWith(s.db).QueryRowContext(ctx))
	switch err {
	case nil:
		return &rn, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *pgSQLRoster) DeleteRosterNotification(ctx context.Context, contact, jid string) error {
	q := sq.Delete("roster_notifications").Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *pgSQLRoster) FetchRosterGroups(ctx context.Context, username string) ([]string, error) {
	q := sq.Select("`group`").
		From("roster_groups").
		Where(sq.Eq{"username": username}).
		GroupBy("`group`")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func scanRosterNotificationEntity(rn *rostermodel.Notification, scanner rowScanner) error {
	var presenceXML string
	if err := scanner.Scan(&rn.Contact, &rn.JID, &presenceXML); err != nil {
		return err
	}
	parser := xmpp.NewParser(strings.NewReader(presenceXML), xmpp.DefaultMode, 0)
	elem, err := parser.ParseElement()
	if err != nil {
		return err
	}
	fromJID, _ := jid.NewWithString(elem.From(), true)
	toJID, _ := jid.NewWithString(elem.To(), true)
	rn.Presence, _ = xmpp.NewPresenceFromElement(elem, fromJID, toJID)
	return nil
}

func scanRosterItemEntity(ri *rostermodel.Item, scanner rowScanner) error {
	var groupsBytes string
	if err := scanner.Scan(&ri.Username, &ri.JID, &ri.Name, &ri.Subscription, &groupsBytes, &ri.Ask, &ri.Ver); err != nil {
		return err
	}
	if len(groupsBytes) > 0 {
		if err := json.NewDecoder(strings.NewReader(groupsBytes)).Decode(&ri.Groups); err != nil {
			return err
		}
	}
	return nil
}

func scanRosterItemEntities(scanner rowsScanner) ([]rostermodel.Item, error) {
	var ret []rostermodel.Item
	for scanner.Next() {
		var ri rostermodel.Item
		if err := scanRosterItemEntity(&ri, scanner); err != nil {
			return nil, err
		}
		ret = append(ret, ri)
	}
	return ret, nil
}

func fetchRosterVer(ctx context.Context, username string, runner sq.BaseRunner) (rostermodel.Version, error) {
	q := sq.Select("COALESCE(MAX(ver), 0)", "COALESCE(MAX(last_deletion_ver), 0)").
		From("roster_versions").
		Where(sq.Eq{"username": username})

	var ver rostermodel.Version
	row := q.RunWith(runner).QueryRowContext(ctx)
	err := row.Scan(&ver.Ver, &ver.DeletionVer)
	switch err {
	case nil:
		return ver, nil
	default:
		return rostermodel.Version{}, err
	}
}
