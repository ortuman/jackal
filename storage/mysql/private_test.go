/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")
	rawXML := private.String()

	s, mock := newPrivateMock()
	mock.ExpectExec("INSERT INTO private_storage (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "exodus:ns", rawXML, rawXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newPrivateMock()
	mock.ExpectExec("INSERT INTO private_storage (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "exodus:ns", rawXML, rawXML).
		WillReturnError(errMySQLStorage)

	err = s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchPrivateXML(t *testing.T) {
	var privateColumns = []string{"data"}

	s, mock := newPrivateMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow("<exodus xmlns='exodus:ns'><stuff/></exodus>"))

	elems, err := s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(elems))

	s, mock = newPrivateMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow("<exodus xmlns='exodus:ns'><stuff/>"))

	elems, err = s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = newPrivateMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow(""))

	elems, err = s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = newPrivateMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns))

	elems, err = s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = newPrivateMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnError(errMySQLStorage)

	elems, err = s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
	require.Equal(t, 0, len(elems))
}

func newPrivateMock() (*mySQLPrivate, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLPrivate{
		mySQLStorage: s,
		pool:         pool.NewBufferPool(),
	}, sqlMock
}
