/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	sq "github.com/Masterminds/squirrel"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// UpsertRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func (s *Storage) UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) (rostermodel.Version, error) {
	var ver rostermodel.Version

	err := s.inTransactionTx(ctx, func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username", "created_at", "updated_at").
			Values(ri.Username, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE ver = ver + 1, updated_at = NOW()")
		if _, err := q.RunWith(tx).ExecContext(ctx); err != nil {
			return err
		}
		groupsBytes, err := json.Marshal(ri.Groups)
		if err != nil {
			return err
		}

		verExpr := sq.Expr("(SELECT ver FROM roster_versions WHERE username = ?)", ri.Username)
		q = sq.Insert("roster_items").
			Columns("username", "jid", "name", "subscription", "`groups`", "ask", "ver", "created_at", "updated_at").
			Values(ri.Username, ri.JID, ri.Name, ri.Subscription, groupsBytes, ri.Ask, verExpr, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE name = ?, subscription = ?, `groups` = ?, ask = ?, ver = ver + 1, updated_at = NOW()", ri.Name, ri.Subscription, groupsBytes, ri.Ask)
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
				Columns("username", "jid", "`group`", "created_at", "updated_at").
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

// DeleteRosterItem deletes a roster item entity from storage.
func (s *Storage) DeleteRosterItem(ctx context.Context, username, jid string) (rostermodel.Version, error) {
	var ver rostermodel.Version

	err := s.inTransactionTx(ctx, func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username", "created_at", "updated_at").
			Values(username, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE ver = ver + 1, last_deletion_ver = ver, updated_at = NOW()")

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

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func (s *Storage) FetchRosterItems(ctx context.Context, username string) ([]rostermodel.Item, rostermodel.Version, error) {
	q := sq.Select("username", "jid", "name", "subscription", "`groups`", "ask", "ver").
		From("roster_items").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at DESC")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	defer func() { _ = rows.Close() }()

	items, err := s.scanRosterItemEntities(rows)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	ver, err := fetchRosterVer(ctx, username, s.db)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	return items, ver, nil
}

// FetchRosterItemsInGroups retrieves from storage all roster item entities
// associated to a given user and a set of groups.
func (s *Storage) FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	q := sq.Select("ris.username", "ris.jid", "ris.name", "ris.subscription", "ris.`groups`", "ris.ask", "ris.ver").
		From("roster_items ris").
		LeftJoin("roster_groups g ON ris.username = g.username").
		Where(sq.And{sq.Eq{"ris.username": username}, sq.Eq{"g.group": groups}}).
		OrderBy("ris.created_at DESC")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	defer func() { _ = rows.Close() }()

	items, err := s.scanRosterItemEntities(rows)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	ver, err := fetchRosterVer(ctx, username, s.db)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	return items, ver, nil
}

// FetchRosterItem retrieves from storage a roster item entity.
func (s *Storage) FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error) {
	q := sq.Select("username", "jid", "name", "subscription", "`groups`", "ask", "ver").
		From("roster_items").
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}})

	var ri rostermodel.Item
	err := s.scanRosterItemEntity(&ri, q.RunWith(s.db).QueryRowContext(ctx))
	switch err {
	case nil:
		return &ri, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

// UpsertRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func (s *Storage) UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error {
	presenceXML := rn.Presence.String()
	q := sq.Insert("roster_notifications").
		Columns("contact", "jid", "elements", "updated_at", "created_at").
		Values(rn.Contact, rn.JID, presenceXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE elements = ?, updated_at = NOW()", presenceXML)
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func (s *Storage) FetchRosterNotifications(ctx context.Context, contact string) ([]rostermodel.Notification, error) {
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
		if err := s.scanRosterNotificationEntity(&rn, rows); err != nil {
			return nil, err
		}
		ret = append(ret, rn)
	}
	return ret, nil
}

// FetchRosterNotification retrieves from storage a roster notification entity.
func (s *Storage) FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	q := sq.Select("contact", "jid", "elements").
		From("roster_notifications").
		Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})

	var rn rostermodel.Notification
	err := s.scanRosterNotificationEntity(&rn, q.RunWith(s.db).QueryRowContext(ctx))
	switch err {
	case nil:
		return &rn, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

// DeleteRosterNotification deletes a roster notification entity from storage.
func (s *Storage) DeleteRosterNotification(ctx context.Context, contact, jid string) error {
	q := sq.Delete("roster_notifications").Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

// FetchRosterGroups retrieves all groups associated to a user roster
func (s *Storage) FetchRosterGroups(ctx context.Context, username string) ([]string, error) {
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

func (s *Storage) scanRosterNotificationEntity(rn *rostermodel.Notification, scanner rowScanner) error {
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

func (s *Storage) scanRosterItemEntity(ri *rostermodel.Item, scanner rowScanner) error {
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

func (s *Storage) scanRosterItemEntities(scanner rowsScanner) ([]rostermodel.Item, error) {
	var ret []rostermodel.Item
	for scanner.Next() {
		var ri rostermodel.Item
		if err := s.scanRosterItemEntity(&ri, scanner); err != nil {
			return nil, err
		}
		ret = append(ret, ri)
	}
	return ret, nil
}

func fetchRosterVer(ctx context.Context, username string, runner sq.BaseRunner) (rostermodel.Version, error) {
	q := sq.Select("IFNULL(MAX(ver), 0)", "IFNULL(MAX(last_deletion_ver), 0)").
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
