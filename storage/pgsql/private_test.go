/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestInsertPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")
	rawXML := private.String()

	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO private_storage (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs("ortuman", "exodus:ns", rawXML, rawXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdatePrivateXML([]xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO private_storage (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs("ortuman", "exodus:ns", rawXML, rawXML).
		WillReturnError(errGeneric)

	err = s.InsertOrUpdatePrivateXML([]xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestFetchPrivateXML(t *testing.T) {
	var privateColumns = []string{"data"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow("<exodus xmlns='exodus:ns'><stuff/></exodus>"))

	elems, err := s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(elems))

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow("<exodus xmlns='exodus:ns'><stuff/>"))

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow(""))

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns))

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnError(errGeneric)

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
	require.Equal(t, 0, len(elems))
}
