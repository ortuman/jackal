/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package storage

import (
	"bytes"
	"database/sql"
	"fmt"

	// driver implementation
	_ "github.com/go-sql-driver/mysql"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/entity"
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

func (s *mySQL) FetchUser(username string) (*entity.User, error) {
	row := s.db.QueryRow("SELECT username, password FROM users WHERE username = ?", username)
	u := entity.User{}
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

func (s *mySQL) InsertOrUpdateUser(u *entity.User) error {
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

func (s *mySQL) FetchVCard(username string) (*xml.Element, error) {
	row := s.db.QueryRow("SELECT vcard FROM vcards WHERE username = ?", username)
	var vCard string
	err := row.Scan(&vCard)
	switch err {
	case nil:
		/*
			parser := xml.NewParser()
			parser.ParseElements(strings.NewReader(vCard))
			return parser.PopElement(), nil
		*/
		return nil, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQL) InsertOrUpdateVCard(vCard *xml.Element, username string) error {
	stmt := `` +
		`INSERT INTO vcards(username, vcard, updated_at, created_at)` +
		`VALUES(?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE vcard = ?, updated_at = NOW()`
	rawXML := vCard.XML(true)
	_, err := s.db.Exec(stmt, username, rawXML, rawXML)
	return err
}

func (s *mySQL) FetchPrivateXML(namespace string, username string) ([]*xml.Element, error) {
	row := s.db.QueryRow("SELECT data FROM private_storage WHERE username = ? AND namespace = ?", username, namespace)
	var privateXML string
	err := row.Scan(&privateXML)
	switch err {
	case nil:
		/*
			parser := xml.NewParser()
			parser.ParseElements(strings.NewReader(fmt.Sprintf("<root>%s</root>", privateXML)))
			rootEl := parser.PopElement()
			if rootEl != nil {
				return rootEl.Elements(), nil
			}
			fallthrough
		*/
		return nil, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQL) InsertOrUpdatePrivateXML(privateXML []*xml.Element, namespace string, username string) error {
	stmt := `` +
		`INSERT INTO private_storage(username, namespace, data, updated_at, created_at)` +
		`VALUES(?, ?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE data = ?, updated_at = NOW()`

	buf := new(bytes.Buffer)
	for _, elem := range privateXML {
		buf.WriteString(elem.XML(true))
	}
	rawXML := buf.String()
	_, err := s.db.Exec(stmt, username, namespace, rawXML, rawXML)
	return err
}

func (s *mySQL) InsertOfflineMessage(message *xml.Element, username string) error {
	stmt := `INSERT INTO offline_messages(username, data, created_at) VALUES(?, ?, NOW())`
	_, err := s.db.Exec(stmt, username, message.XML(true))
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

func (s *mySQL) FetchOfflineMessages(username string) ([]*xml.Element, error) {
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

	/*
		parser := xml.NewParser()
		parser.ParseElements(bytes.NewReader(buf.Bytes()))
		rootEl := parser.PopElement()
		if rootEl == nil {
			return []*xml.Element{}, nil
		}
		return rootEl.Elements(), nil
	*/
	return []*xml.Element{}, nil
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
