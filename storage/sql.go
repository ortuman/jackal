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
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	sq "gopkg.in/Masterminds/squirrel.v1"
)

var (
	nowExpr = sq.Expr("NOW()")
)

type mySQLStorage struct {
	db     *sql.DB
	pool   *pool.BufferPool
	doneCh chan chan bool
}

func newMySQLStorage(cfg *config.MySQLDb) *mySQLStorage {
	var err error
	s := &mySQLStorage{
		pool:   pool.NewBufferPool(),
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
	s := &mySQLStorage{
		pool: pool.NewBufferPool(),
	}
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
	q := sq.Insert("users").
		Columns("username", "password", "updated_at", "created_at").
		Values(u.Username, u.Password, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE password = ?, updated = NOW()", u.Password)

	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *mySQLStorage) FetchUser(username string) (*model.User, error) {
	q := sq.Select("username", "password").From("users").Where(sq.Eq{"username": username})

	var usr model.User
	err := q.RunWith(s.db).QueryRow().Scan(&usr.Username, &usr.Password)
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
	return s.inTransaction(func(tx *sql.Tx) error {
		var err error
		_, err = sq.Delete("offline_messages").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("roster_items").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("roster_versions").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("private_storage").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("vcards").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("users").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *mySQLStorage) UserExists(username string) (bool, error) {
	q := sq.Select("COUNT(*)").From("users").Where(sq.Eq{"username": username})

	var count int
	err := q.RunWith(s.db).QueryRow().Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}

func (s *mySQLStorage) InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error) {
	err := s.inTransaction(func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username", "created_at", "updated_at").
			Values(ri.User, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE ver = ver + 1, updated_at = NOW()")

		if _, err := q.RunWith(tx).Exec(); err != nil {
			return err
		}
		groups := strings.Join(ri.Groups, ";")

		verExpr := sq.Expr("(SELECT ver FROM roster_versions WHERE username = ?)", ri.User)
		q = sq.Insert("roster_items").
			Columns("user", "contact", "name", "subscription", "groups", "ask", "ver", "created_at", "updated_at").
			Values(ri.User, ri.Contact, ri.Name, ri.Subscription, groups, ri.Ask, verExpr, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE name = ?, subscription = ?, groups = ?, ask = ?, ver = ver + 1, updated_at = NOW()", ri.Name, ri.Subscription, groups, ri.Ask)

		_, err := q.RunWith(tx).Exec()
		return err
	})
	if err != nil {
		return model.RosterVersion{}, err
	}
	return s.fetchRosterVer(ri.User)
}

func (s *mySQLStorage) DeleteRosterItem(user, contact string) (model.RosterVersion, error) {
	err := s.inTransaction(func(tx *sql.Tx) error {
		q := sq.Insert("roster_versions").
			Columns("username", "created_at", "updated_at").
			Values(user, nowExpr, nowExpr).
			Suffix("ON DUPLICATE KEY UPDATE ver = ver + 1, last_deletion_ver = ver, updated_at = NOW()")

		if _, err := q.RunWith(tx).Exec(); err != nil {
			return err
		}
		_, err := sq.Delete("roster_items").
			Where(sq.And{sq.Eq{"user": user}, sq.Eq{"contact": contact}}).
			RunWith(tx).Exec()
		return err
	})
	if err != nil {
		return model.RosterVersion{}, err
	}
	return s.fetchRosterVer(user)
}

func (s *mySQLStorage) FetchRosterItems(user string) ([]model.RosterItem, model.RosterVersion, error) {
	q := sq.Select("user", "contact", "name", "subscription", "groups", "ask", "ver").
		From("roster_items").
		Where(sq.Eq{"user": user}).
		OrderBy("created_at DESC")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, model.RosterVersion{}, err
	}
	defer rows.Close()

	items, err := scanRosterItemEntities(rows)
	if err != nil {
		return nil, model.RosterVersion{}, err
	}
	ver, err := s.fetchRosterVer(user)
	return items, ver, nil
}

func (s *mySQLStorage) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	q := sq.Select("user", "contact", "name", "subscription", "groups", "ask", "ver").
		From("roster_items").
		Where(sq.And{sq.Eq{"user": user}, sq.Eq{"contact": contact}})

	var ri model.RosterItem
	err := scanRosterItemEntity(&ri, q.RunWith(s.db).QueryRow())
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
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	for _, elem := range rn.Elements {
		buf.WriteString(elem.String())
	}
	elementsXML := buf.String()

	q := sq.Insert("roster_notifications").
		Columns("user", "contact", "elements", "updated_at", "created_at").
		Values(rn.User, rn.Contact, elementsXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE elements = ?, updated_at = NOW()", elementsXML)
	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *mySQLStorage) DeleteRosterNotification(user, contact string) error {
	q := sq.Delete("roster_notifications").Where(sq.And{sq.Eq{"user": user}, sq.Eq{"contact": contact}})
	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *mySQLStorage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	q := sq.Select("user", "contact", "elements").
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

func (s *mySQLStorage) InsertOrUpdateVCard(vCard xml.XElement, username string) error {
	rawXML := vCard.String()
	q := sq.Insert("vcards").
		Columns("username", "vcard", "updated_at", "created_at").
		Values(username, rawXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE vcard = ?, updated_at = NOW()", rawXML)

	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *mySQLStorage) FetchVCard(username string) (xml.XElement, error) {
	q := sq.Select("vcard").From("vcards").Where(sq.Eq{"username": username})

	var vCard string
	err := q.RunWith(s.db).QueryRow().Scan(&vCard)
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

func (s *mySQLStorage) InsertOrUpdatePrivateXML(privateXML []xml.XElement, namespace string, username string) error {
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	for _, elem := range privateXML {
		elem.ToXML(buf, true)
	}
	rawXML := buf.String()

	q := sq.Insert("private_storage").
		Columns("username", "namespace", "data", "updated_at", "created_at").
		Values(username, namespace, rawXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE data = ?, updated_at = NOW()", rawXML)

	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *mySQLStorage) FetchPrivateXML(namespace string, username string) ([]xml.XElement, error) {
	q := sq.Select("data").
		From("private_storage").
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"namespace": namespace}})

	var privateXML string
	err := q.RunWith(s.db).QueryRow().Scan(&privateXML)
	switch err {
	case nil:
		buf := s.pool.Get()
		defer s.pool.Put(buf)
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

func (s *mySQLStorage) InsertOfflineMessage(message xml.XElement, username string) error {
	q := sq.Insert("offline_messages").
		Columns("username", "data", "created_at").
		Values(username, message.String(), nowExpr)
	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *mySQLStorage) CountOfflineMessages(username string) (int, error) {
	q := sq.Select("COUNT(*)").
		From("offline_messages").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	var count int
	err := q.RunWith(s.db).Scan(&count)
	switch err {
	case nil:
		return count, nil
	default:
		return 0, err
	}
}

func (s *mySQLStorage) FetchOfflineMessages(username string) ([]xml.XElement, error) {
	q := sq.Select("data").
		From("offline_messages").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buf := s.pool.Get()
	defer s.pool.Put(buf)

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
	q := sq.Delete("offline_messages").Where(sq.Eq{"username": username})
	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *mySQLStorage) fetchRosterVer(username string) (model.RosterVersion, error) {
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
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
