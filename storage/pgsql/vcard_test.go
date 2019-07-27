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

func TestInsertVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	rawXML := vCard.String()

	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO vcards (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs("ortuman", rawXML, rawXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.UpsertVCard(vCard, "ortuman")
	require.Nil(t, err)
	require.NotNil(t, vCard)
	require.Nil(t, mock.ExpectationsWereMet())

	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO vcards (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs("ortuman", rawXML, rawXML).
		WillReturnError(errGeneric)

	err = s.UpsertVCard(vCard, "ortuman")
	require.Equal(t, errGeneric, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestFetchVCard(t *testing.T) {
	var vCardColumns = []string{"vcard"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(vCardColumns).AddRow("<vCard><FN>Miguel Ángel</FN></vCard>"))

	vCard, err := s.FetchVCard("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, vCard)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(vCardColumns))

	vCard, err = s.FetchVCard("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Nil(t, vCard)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnError(errGeneric)

	vCard, _ = s.FetchVCard("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, vCard)
}
