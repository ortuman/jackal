/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertOfflineMessages(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	message := xmpp.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xmpp.NewElementName("body"))
	m, _ := xmpp.NewMessageFromElement(message, j, j)
	messageXML := m.String()

	s, mock := newOfflineMock()
	mock.ExpectExec("INSERT INTO offline_messages (.+)").
		WithArgs("ortuman", messageXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOfflineMessage(context.Background(), m, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newOfflineMock()
	mock.ExpectExec("INSERT INTO offline_messages (.+)").
		WithArgs("ortuman", messageXML).
		WillReturnError(errMySQLStorage)

	err = s.InsertOfflineMessage(context.Background(), m, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
}

func TestMySQLStorageCountOfflineMessages(t *testing.T) {
	countColums := []string{"count"}

	s, mock := newOfflineMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countColums).AddRow(1))

	cnt, _ := s.CountOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 1, cnt)

	s, mock = newOfflineMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countColums))

	cnt, _ = s.CountOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 0, cnt)

	s, mock = newOfflineMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, err := s.CountOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchOfflineMessages(t *testing.T) {
	var offlineMessagesColumns = []string{"data"}

	s, mock := newOfflineMock()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(offlineMessagesColumns).AddRow("<message id='abc'><body>Hi!</body></message>"))

	msgs, _ := s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 1, len(msgs))

	s, mock = newOfflineMock()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(offlineMessagesColumns))

	msgs, _ = s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 0, len(msgs))

	s, mock = newOfflineMock()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(offlineMessagesColumns).AddRow("<message id='abc'><body>Hi!"))

	_, err := s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)

	s, mock = newOfflineMock()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageDeleteOfflineMessages(t *testing.T) {
	s, mock := newOfflineMock()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newOfflineMock()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnError(errMySQLStorage)

	err = s.DeleteOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func newOfflineMock() (*mySQLOffline, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLOffline{
		mySQLStorage: s,
		pool:         pool.NewBufferPool(),
	}, sqlMock
}
