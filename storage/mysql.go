/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"

	// SQL driver implementation
	_ "github.com/go-sql-driver/mysql"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xml"
)

const maxTransactionRetries = 4

type mySQL struct {
	db *sql.DB
}

func newMySQLStorage() storage {
	s := &mySQL{}
	host := config.DefaultConfig.Storage.MySQL.Host
	user := config.DefaultConfig.Storage.MySQL.User
	pass := config.DefaultConfig.Storage.MySQL.Password
	db := config.DefaultConfig.Storage.MySQL.Database
	poolSize := config.DefaultConfig.Storage.MySQL.PoolSize

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, pass, host, db)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = conn.Ping()
	if err != nil {
		log.Fatalf("%v", err)
	}

	// set max opened connection count
	conn.SetMaxOpenConns(poolSize)

	s.db = conn
	return s
}

func (s *mySQL) FetchUser(username string) (*User, error) {
	row := s.db.QueryRow("SELECT username, password FROM users WHERE username = ?", username)
	u := User{}
	err := row.Scan(&u.Username, &u.Password)
	switch err {
	case nil:
		return &u, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQL) InsertOrUpdateUser(u *User) error {
	stmt := `` +
		`INSERT INTO users(username, password, updated_at, created_at)` +
		`VALUES(?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE password = ?, updated_at = NOW()`
	_, err := s.db.Exec(stmt, u.Username, u.Password, u.Password)
	return err
}

func (s *mySQL) DeleteUser(username string) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		var err error
		_, err = tx.Exec("DELETE FROM offline_messages WHERE username = ?", username)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM roster_items WHERE username = ?", username)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM private_storage WHERE username = ?", username)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM vcards WHERE username = ?", username)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM users WHERE username = ?", username)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *mySQL) UserExists(username string) (bool, error) {
	row := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username)
	var count int
	err := row.Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}

func (s *mySQL) InsertOrUpdateRosterItem(ri *RosterItem) error {
	groups := strings.Join(ri.Groups, ";")
	params := []interface{}{
		ri.User,
		ri.Contact,
		ri.Domain,
		ri.Name,
		ri.Subscription,
		groups,
		ri.Ask,
		ri.Name,
		ri.Subscription,
		groups,
		ri.Ask,
	}
	stmt := `` +
		`INSERT INTO roster_items(user, contact, domain, name, subscription, groups, ask, updated_at, created_at)` +
		`VALUES(?, ?, ?, ?, ?, ?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE name = ?, subscription = ?, groups = ?, ask = ?, updated_at = NOW()`
	_, err := s.db.Exec(stmt, params...)
	return err
}

func (s *mySQL) DeleteRosterItem(user, contact, domain string) error {
	stmt := "DELETE FROM roster_items WHERE user = ? AND contact = ? AND domain = ?"
	_, err := s.db.Exec(stmt, user, contact, domain)
	return err
}

func (s *mySQL) FetchInboundRosterItem(user, contact, domain string) (*RosterItem, error) {
	stmt := `` +
		`SELECT user, contact, domain, name, subscription, groups, ask` +
		` FROM roster_items WHERE user = ? AND contact = ? AND domain = ?`
	rows, err := s.db.Query(stmt, user, contact, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return s.rosterItemFromRows(rows)
	}
	return nil, nil
}

func (s *mySQL) FetchInboundRosterItems(user string) ([]RosterItem, error) {
	stmt := `` +
		`SELECT user, contact, domain, name, subscription, groups, ask` +
		` FROM roster_items WHERE  user = ?` +
		` ORDER BY created_at DESC`

	rows, err := s.db.Query(stmt, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.rosterItemsFromRows(rows)
}

func (s *mySQL) InsertOrUpdateRosterNotification(rn *RosterNotification) error {
	stmt := `` +
		`INSERT INTO roster_notifications(user, domain, contact, elements, updated_at, created_at)` +
		`VALUES(?, ?, ?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE elements = ?, updated_at = NOW()`

	buf := new(bytes.Buffer)
	for _, elem := range rn.Elements {
		buf.WriteString(elem.String())
	}
	elementsXML := buf.String()
	_, err := s.db.Exec(stmt, rn.User, rn.Domain, rn.Contact, elementsXML, elementsXML)
	return err
}

func (s *mySQL) DeleteRosterNotification(user, contact string) error {
	_, err := s.db.Exec("DELETE FROM roster_notifications WHERE user = ? AND contact = ?", user, contact)
	return err
}

func (s *mySQL) FetchRosterNotifications(contact string) ([]RosterNotification, error) {
	stmt := `SELECT user, domain, contact, elements FROM roster_notifications WHERE contact = ? ORDER BY created_at`
	rows, err := s.db.Query(stmt, contact)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buf := new(bytes.Buffer)

	var ret []RosterNotification
	for rows.Next() {
		var rn RosterNotification
		var notificationXML string
		rows.Scan(&rn.User, &rn.Domain, &rn.Contact, &notificationXML)
		buf.Reset()
		buf.WriteString("<root>")
		buf.WriteString(notificationXML)
		buf.WriteString("</root>")

		parser := xml.NewParser(buf)
		root, err := parser.ParseElement()
		if err != nil {
			return nil, err
		}
		rn.Elements = root.Elements()

		ret = append(ret, rn)
	}
	return ret, nil
}

func (s *mySQL) FetchVCard(username string) (xml.Element, error) {
	row := s.db.QueryRow("SELECT vcard FROM vcards WHERE username = ?", username)
	var vCard string
	err := row.Scan(&vCard)
	switch err {
	case nil:
		parser := xml.NewParser(strings.NewReader(vCard))
		return parser.ParseElement()
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQL) InsertOrUpdateVCard(vCard xml.Element, username string) error {
	stmt := `` +
		`INSERT INTO vcards(username, vcard, updated_at, created_at)` +
		`VALUES(?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE vcard = ?, updated_at = NOW()`

	rawXML := vCard.String()
	_, err := s.db.Exec(stmt, username, rawXML, rawXML)
	return err
}

func (s *mySQL) FetchPrivateXML(namespace string, username string) ([]xml.Element, error) {
	row := s.db.QueryRow("SELECT data FROM private_storage WHERE username = ? AND namespace = ?", username, namespace)
	var privateXML string
	err := row.Scan(&privateXML)
	switch err {
	case nil:
		reader := strings.NewReader(fmt.Sprintf("<root>%s</root>", privateXML))
		parser := xml.NewParser(reader)
		rootEl, err := parser.ParseElement()
		if err != nil {
			return nil, err
		} else if rootEl != nil {
			return rootEl.Elements(), nil
		}
		fallthrough
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQL) InsertOrUpdatePrivateXML(privateXML []xml.Element, namespace string, username string) error {
	stmt := `` +
		`INSERT INTO private_storage(username, namespace, data, updated_at, created_at)` +
		`VALUES(?, ?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE data = ?, updated_at = NOW()`

	buf := new(bytes.Buffer)
	for _, elem := range privateXML {
		elem.ToXML(buf, true)
	}
	rawXML := buf.String()
	_, err := s.db.Exec(stmt, username, namespace, rawXML, rawXML)
	return err
}

func (s *mySQL) InsertOfflineMessage(message xml.Element, username string) error {
	stmt := `INSERT INTO offline_messages(username, data, created_at) VALUES(?, ?, NOW())`
	_, err := s.db.Exec(stmt, username, message.String())
	return err
}

func (s *mySQL) CountOfflineMessages(username string) (int, error) {
	row := s.db.QueryRow("SELECT COUNT(*) FROM offline_messages WHERE username = ? ORDER BY created_at", username)
	var count int
	err := row.Scan(&count)
	switch err {
	case nil:
		return count, nil
	default:
		return 0, err
	}
}

func (s *mySQL) FetchOfflineMessages(username string) ([]xml.Element, error) {
	rows, err := s.db.Query("SELECT data FROM offline_messages WHERE username = ? ORDER BY created_at", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buf := bytes.NewBufferString("<root>")
	for rows.Next() {
		var msg string
		rows.Scan(&msg)
		buf.WriteString(msg)
	}
	buf.WriteString("</root>")

	parser := xml.NewParser(bytes.NewReader(buf.Bytes()))
	rootEl, err := parser.ParseElement()
	if err != nil {
		return nil, err
	} else if rootEl == nil {
		return nil, nil
	}
	return rootEl.Elements(), nil
}

func (s *mySQL) DeleteOfflineMessages(username string) error {
	_, err := s.db.Exec("DELETE FROM offline_messages WHERE username = ?", username)
	return err
}

func (s *mySQL) inTransaction(f func(tx *sql.Tx) error) error {
	var err error
	for i := 0; i < maxTransactionRetries; i++ {
		tx, txErr := s.db.Begin()
		if txErr != nil {
			return txErr
		}
		err = f(tx)
		if err != nil {
			tx.Rollback()
			continue
		}
		tx.Commit()
	}
	return err
}

func (s *mySQL) rosterItemsFromRows(rows *sql.Rows) ([]RosterItem, error) {
	var result []RosterItem
	for rows.Next() {
		ri, err := s.rosterItemFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *ri)
	}
	return result, nil
}

func (s *mySQL) rosterItemFromRows(rows *sql.Rows) (*RosterItem, error) {
	var ri RosterItem
	var groups string

	rows.Scan(&ri.User, &ri.Contact, &ri.Domain, &ri.Name, &ri.Subscription, &groups, &ri.Ask)
	ri.Groups = strings.Split(groups, ";")
	return &ri, nil
}
