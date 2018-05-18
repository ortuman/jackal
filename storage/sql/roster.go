/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
)

func (s *Storage) InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error) {
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
			Columns("username", "jid", "name", "subscription", "groups", "ask", "ver", "created_at", "updated_at").
			Values(ri.Username, ri.JID, ri.Name, ri.Subscription, groups, ri.Ask, verExpr, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE name = ?, subscription = ?, groups = ?, ask = ?, ver = ver + 1, updated_at = NOW()", ri.Name, ri.Subscription, groups, ri.Ask)

		_, err := q.RunWith(tx).Exec()
		return err
	})
	if err != nil {
		return model.RosterVersion{}, err
	}
	return s.fetchRosterVer(ri.Username)
}

func (s *Storage) DeleteRosterItem(username, jid string) (model.RosterVersion, error) {
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
		return model.RosterVersion{}, err
	}
	return s.fetchRosterVer(username)
}

func (s *Storage) FetchRosterItems(username string) ([]model.RosterItem, model.RosterVersion, error) {
	q := sq.Select("username", "jid", "name", "subscription", "groups", "ask", "ver").
		From("roster_items").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at DESC")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, model.RosterVersion{}, err
	}
	defer rows.Close()

	items, err := s.scanRosterItemEntities(rows)
	if err != nil {
		return nil, model.RosterVersion{}, err
	}
	ver, err := s.fetchRosterVer(username)
	if err != nil {
		return nil, model.RosterVersion{}, err
	}
	return items, ver, nil
}

func (s *Storage) FetchRosterItem(username, jid string) (*model.RosterItem, error) {
	q := sq.Select("username", "jid", "name", "subscription", "groups", "ask", "ver").
		From("roster_items").
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"jid": jid}})

	var ri model.RosterItem
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

func (s *Storage) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	for _, elem := range rn.Elements {
		buf.WriteString(elem.String())
	}
	elementsXML := buf.String()

	q := sq.Insert("roster_notifications").
		Columns("contact", "jid", "elements", "updated_at", "created_at").
		Values(rn.Contact, rn.JID, elementsXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE elements = ?, updated_at = NOW()", elementsXML)
	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *Storage) DeleteRosterNotification(contact, jid string) error {
	q := sq.Delete("roster_notifications").Where(sq.And{sq.Eq{"contact": contact}, sq.Eq{"jid": jid}})
	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *Storage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	q := sq.Select("contact", "jid", "elements").
		From("roster_notifications").
		Where(sq.Eq{"contact": contact}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buf := s.pool.Get()
	defer s.pool.Put(buf)

	var ret []model.RosterNotification
	for rows.Next() {
		var rn model.RosterNotification
		var notificationXML string
		rows.Scan(&rn.Contact, &rn.JID, &notificationXML)
		buf.Reset()
		buf.WriteString("<root>")
		buf.WriteString(notificationXML)
		buf.WriteString("</root>")

		parser := xml.NewParser(buf, 0)
		root, err := parser.ParseElement()
		if err != nil {
			return nil, err
		}
		rn.Elements = root.Elements().All()

		ret = append(ret, rn)
	}
	return ret, nil
}

func (s *Storage) fetchRosterVer(username string) (model.RosterVersion, error) {
	q := sq.Select("IFNULL(MAX(ver), 0)", "IFNULL(MAX(last_deletion_ver), 0)").
		From("roster_versions").
		Where(sq.Eq{"username": username})

	var ver model.RosterVersion
	row := q.RunWith(s.db).QueryRow()
	err := row.Scan(&ver.Ver, &ver.DeletionVer)
	switch err {
	case nil:
		return ver, nil
	default:
		return model.RosterVersion{}, err
	}
}

func (s *Storage) scanRosterItemEntity(ri *model.RosterItem, scanner rowScanner) error {
	var groups string
	if err := scanner.Scan(&ri.Username, &ri.JID, &ri.Name, &ri.Subscription, &groups, &ri.Ask, &ri.Ver); err != nil {
		return err
	}
	ri.Groups = strings.Split(groups, ";")
	return nil
}

func (s *Storage) scanRosterItemEntities(scanner rowsScanner) ([]model.RosterItem, error) {
	var ret []model.RosterItem
	for scanner.Next() {
		var ri model.RosterItem
		if err := s.scanRosterItemEntity(&ri, scanner); err != nil {
			return nil, err
		}
		ret = append(ret, ri)
	}
	return ret, nil
}
