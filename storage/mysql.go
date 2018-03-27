/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/go-sql-driver/mysql" // SQL driver
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
)

type mySQLStorage struct {
	db     *sql.DB
	doneCh chan chan bool
}

func newMySQLStorage(cfg *config.MySQLDb) *mySQLStorage {
	var err error
	s := &mySQLStorage{
		doneCh: make(chan chan bool),
	}
	host := cfg.Host
	user := cfg.User
	pass := cfg.Password
	db := cfg.Database
	poolSize := cfg.PoolSize

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, pass, host, db)
	s.db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%v", err)
	}
	s.db.SetMaxOpenConns(poolSize) // set max opened connection count

	if err := s.db.Ping(); err != nil {
		log.Fatalf("%v", err)
	}
	go s.loop()

	return s
}

func newMockMySQLStorage() (*mySQLStorage, sqlmock.Sqlmock) {
	var err error
	var sqlMock sqlmock.Sqlmock
	s := &mySQLStorage{}
	s.db, sqlMock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return s, sqlMock
}

func (s *mySQLStorage) Shutdown() {
	ch := make(chan bool)
	s.doneCh <- ch
	<-ch
}

func (s *mySQLStorage) InsertOrUpdateUser(u *model.User) error {
	stmt := `` +
		`INSERT INTO users (username, password, updated_at, created_at)` +
		` VALUES(?, ?, NOW(), NOW())` +
		` ON DUPLICATE KEY UPDATE password = ?, updated_at = NOW()`
	_, err := s.db.Exec(stmt, u.Username, u.Password, u.Password)
	return err
}

func (s *mySQLStorage) FetchUser(username string) (*model.User, error) {
	row := s.db.QueryRow("SELECT username, password FROM users WHERE username = ?", username)

	var usr model.User
	err := row.Scan(&usr.Username, &usr.Password)
	switch err {
	case nil:
		return &usr, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQLStorage) DeleteUser(username string) error {
	stmts := []string{
		"DELETE FROM offline_messages WHERE username = ?",
		"DELETE FROM roster_items WHERE username = ?",
		"DELETE FROM private_storage WHERE username = ?",
		"DELETE FROM vcards WHERE username = ?",
		"DELETE FROM users WHERE username = ?",
	}
	return s.inTransaction(func(tx *sql.Tx) error {
		for _, stmt := range stmts {
			if _, err := tx.Exec(stmt, username); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *mySQLStorage) UserExists(username string) (bool, error) {
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

func (s *mySQLStorage) InsertOrUpdateRosterItem(ri *model.RosterItem) error {
	groups := strings.Join(ri.Groups, ";")
	params := []interface{}{
		ri.User,
		ri.Contact,
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
		`INSERT INTO roster_items (user, contact, name, subscription, groups, ask, updated_at, created_at)` +
		` VALUES(?, ?, ?, ?, ?, ?, NOW(), NOW())` +
		` ON DUPLICATE KEY UPDATE name = ?, subscription = ?, groups = ?, ask = ?, updated_at = NOW()`
	_, err := s.db.Exec(stmt, params...)
	return err
}

func (s *mySQLStorage) DeleteRosterItem(user, contact string) error {
	stmt := "DELETE FROM roster_items WHERE user = ? AND contact = ?"
	_, err := s.db.Exec(stmt, user, contact)
	return err
}

func (s *mySQLStorage) FetchRosterItems(user string) ([]model.RosterItem, error) {
	stmt := `` +
		`SELECT user, contact, name, subscription, groups, ask` +
		` FROM roster_items WHERE  user = ?` +
		` ORDER BY created_at DESC`

	rows, err := s.db.Query(stmt, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRosterItemEntities(rows)
}

func (s *mySQLStorage) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	stmt := `` +
		`SELECT user, contact, name, subscription, groups, ask` +
		` FROM roster_items WHERE user = ? AND contact = ?`
	row := s.db.QueryRow(stmt, user, contact)

	var ri model.RosterItem
	err := scanRosterItemEntity(&ri, row)
	switch err {
	case nil:
		return &ri, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQLStorage) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	stmt := `` +
		`INSERT INTO roster_notifications (user, contact, elements, updated_at, created_at)` +
		` VALUES(?, ?, ?, NOW(), NOW())` +
		` ON DUPLICATE KEY UPDATE elements = ?, updated_at = NOW()`

	buf := pool.Get()
	defer pool.Put(buf)
	for _, elem := range rn.Elements {
		buf.WriteString(elem.String())
	}
	elementsXML := buf.String()
	_, err := s.db.Exec(stmt, rn.User, rn.Contact, elementsXML, elementsXML)
	return err
}

func (s *mySQLStorage) DeleteRosterNotification(user, contact string) error {
	_, err := s.db.Exec("DELETE FROM roster_notifications WHERE user = ? AND contact = ?", user, contact)
	return err
}

func (s *mySQLStorage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	stmt := `SELECT user, contact, elements FROM roster_notifications WHERE contact = ? ORDER BY created_at`
	rows, err := s.db.Query(stmt, contact)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buf := pool.Get()
	defer pool.Put(buf)

	var ret []model.RosterNotification
	for rows.Next() {
		var rn model.RosterNotification
		var notificationXML string
		rows.Scan(&rn.User, &rn.Contact, &notificationXML)
		buf.Reset()
		buf.WriteString("<root>")
		buf.WriteString(notificationXML)
		buf.WriteString("</root>")

		parser := xml.NewParser(buf)
		root, err := parser.ParseElement()
		if err != nil {
			return nil, err
		}
		rn.Elements = root.Elements().All()

		ret = append(ret, rn)
	}
	return ret, nil
}

func (s *mySQLStorage) InsertOrUpdateVCard(vCard xml.Element, username string) error {
	stmt := `` +
		`INSERT INTO vcards (username, vcard, updated_at, created_at)` +
		` VALUES(?, ?, NOW(), NOW())` +
		` ON DUPLICATE KEY UPDATE vcard = ?, updated_at = NOW()`

	rawXML := vCard.String()
	_, err := s.db.Exec(stmt, username, rawXML, rawXML)
	return err
}

func (s *mySQLStorage) FetchVCard(username string) (xml.Element, error) {
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

func (s *mySQLStorage) InsertOrUpdatePrivateXML(privateXML []xml.Element, namespace string, username string) error {
	stmt := `` +
		`INSERT INTO private_storage (username, namespace, data, updated_at, created_at)` +
		` VALUES(?, ?, ?, NOW(), NOW())` +
		` ON DUPLICATE KEY UPDATE data = ?, updated_at = NOW()`

	buf := pool.Get()
	defer pool.Put(buf)
	for _, elem := range privateXML {
		elem.ToXML(buf, true)
	}
	rawXML := buf.String()
	_, err := s.db.Exec(stmt, username, namespace, rawXML, rawXML)
	return err
}

func (s *mySQLStorage) FetchPrivateXML(namespace string, username string) ([]xml.Element, error) {
	row := s.db.QueryRow("SELECT data FROM private_storage WHERE username = ? AND namespace = ?", username, namespace)
	var privateXML string
	err := row.Scan(&privateXML)
	switch err {
	case nil:
		buf := pool.Get()
		defer pool.Put(buf)
		buf.WriteString("<root>")
		buf.WriteString(privateXML)
		buf.WriteString("</root>")

		parser := xml.NewParser(buf)
		rootEl, err := parser.ParseElement()
		if err != nil {
			return nil, err
		}
		return rootEl.Elements().All(), nil

	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQLStorage) InsertOfflineMessage(message xml.Element, username string) error {
	stmt := `INSERT INTO offline_messages (username, data, created_at) VALUES(?, ?, NOW())`
	_, err := s.db.Exec(stmt, username, message.String())
	return err
}

func (s *mySQLStorage) CountOfflineMessages(username string) (int, error) {
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

func (s *mySQLStorage) FetchOfflineMessages(username string) ([]xml.Element, error) {
	rows, err := s.db.Query("SELECT data FROM offline_messages WHERE username = ? ORDER BY created_at", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buf := pool.Get()
	defer pool.Put(buf)

	buf.WriteString("<root>")
	for rows.Next() {
		var msg string
		rows.Scan(&msg)
		buf.WriteString(msg)
	}
	buf.WriteString("</root>")

	parser := xml.NewParser(buf)
	rootEl, err := parser.ParseElement()
	if err != nil {
		return nil, err
	}
	return rootEl.Elements().All(), nil
}

func (s *mySQLStorage) DeleteOfflineMessages(username string) error {
	_, err := s.db.Exec("DELETE FROM offline_messages WHERE username = ?", username)
	return err
}

func (s *mySQLStorage) loop() {
	tc := time.NewTicker(time.Second * 15)
	defer tc.Stop()
	for {
		select {
		case <-tc.C:
			err := s.db.Ping()
			if err != nil {
				log.Error(err)
			}
		case ch := <-s.doneCh:
			s.db.Close()
			close(ch)
			return
		}
	}
}

func (s *mySQLStorage) inTransaction(f func(tx *sql.Tx) error) error {
	tx, txErr := s.db.Begin()
	if txErr != nil {
		return txErr
	}
	err := f(tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
