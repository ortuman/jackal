/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// InsertOrUpdateRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func (s *Storage) InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
	err := s.inTransaction(func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username", "created_at", "updated_at").
			Values(ri.Username, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE ver = ver + 1, updated_at = NOW()")

		if _, err := q.RunWith(tx).Exec(); err != nil {
			return err
		}
		groups := strings.Join(ri.Groups, ";")

		verExpr := sq.Expr("(SELECT ver FROM roster_versions WHERE username = ?)", ri.Username)
		q = sq.Insert("roster_items").
			Columns("username", "jid", "name", "subscription", "`groups`", "ask", "ver", "created_at", "updated_at").
			Values(ri.Username, ri.JID, ri.Name, ri.Subscription, groups, ri.Ask, verExpr, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE name = ?, subscription = ?, `groups` = ?, ask = ?, ver = ver + 1, updated_at = NOW()", ri.Name, ri.Subscription, groups, ri.Ask)

		_, err := q.RunWith(tx).Exec()
		return err
	})
	if err != nil {
		return rostermodel.Version{}, err
	}
	return s.fetchRosterVer(ri.Username)
}

// DeleteRosterItem deletes a roster item entity from storage.
func (s *Storage) DeleteRosterItem(username, jid string) (rostermodel.Version, error) {
	err := s.inTransaction(func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username", "created_at", "updated_at").
			Values(username, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE ver = ver + 1, last_deletion_ver = ver, updated_at = NOW()")

		if _, err := q.RunWith(tx).Exec(); err != nil {
			return err
		}
		_, err := sq.Delete("roster_items").
			Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}}).
			RunWith(tx).Exec()
		return err
	})
	if err != nil {
		return rostermodel.Version{}, err
	}
	return s.fetchRosterVer(username)
}

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func (s *Storage) FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error) {
	q := sq.Select("username", "jid", "name", "subscription", "`groups`", "ask", "ver").
		From("roster_items").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at DESC")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	defer rows.Close()

	items, err := s.scanRosterItemEntities(rows)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	ver, err := s.fetchRosterVer(username)
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	return items, ver, nil
}

// FetchRosterItem retrieves from storage a roster item entity.
func (s *Storage) FetchRosterItem(username, jid string) (*rostermodel.Item, error) {
	q := sq.Select("username", "jid", "name", "subscription", "`groups`", "ask", "ver").
		From("roster_items").
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}})

	var ri rostermodel.Item
	err := s.scanRosterItemEntity(&ri, q.RunWith(s.db).QueryRow())
	switch err {
	case nil:
		return &ri, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

// InsertOrUpdateRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func (s *Storage) InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error {
	presenceXML := rn.Presence.String()
	q := sq.Insert("roster_notifications").
		Columns("contact", "jid", "elements", "updated_at", "created_at").
		Values(rn.Contact, rn.JID, presenceXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE elements = ?, updated_at = NOW()", presenceXML)
	_, err := q.RunWith(s.db).Exec()
	return err
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func (s *Storage) FetchRosterNotifications(contact string) ([]rostermodel.Notification, error) {
	q := sq.Select("contact", "jid", "elements").
		From("roster_notifications").
		Where(sq.Eq{"contact": contact}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
func (s *Storage) FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error) {
	q := sq.Select("contact", "jid", "elements").
		From("roster_notifications").
		Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})

	var rn rostermodel.Notification
	err := s.scanRosterNotificationEntity(&rn, q.RunWith(s.db).QueryRow())
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
func (s *Storage) DeleteRosterNotification(contact, jid string) error {
	q := sq.Delete("roster_notifications").Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})
	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *Storage) fetchRosterVer(username string) (rostermodel.Version, error) {
	q := sq.Select("IFNULL(MAX(ver), 0)", "IFNULL(MAX(last_deletion_ver), 0)").
		From("roster_versions").
		Where(sq.Eq{"username": username})

	var ver rostermodel.Version
	row := q.RunWith(s.db).QueryRow()
	err := row.Scan(&ver.Ver, &ver.DeletionVer)
	switch err {
	case nil:
		return ver, nil
	default:
		return rostermodel.Version{}, err
	}
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
	var groups string
	if err := scanner.Scan(&ri.Username, &ri.JID, &ri.Name, &ri.Subscription, &groups, &ri.Ask, &ri.Ver); err != nil {
		return err
	}
	ri.Groups = strings.Split(groups, ";")
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
