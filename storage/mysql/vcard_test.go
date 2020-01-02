/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	rawXML := vCard.String()

	s, mock := newVCardMock()
	mock.ExpectExec("INSERT INTO vcards (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", rawXML, rawXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.UpsertVCard(context.Background(), vCard, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, vCard)

	s, mock = newVCardMock()
	mock.ExpectExec("INSERT INTO vcards (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", rawXML, rawXML).
		WillReturnError(errMySQLStorage)

	err = s.UpsertVCard(context.Background(), vCard, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchVCard(t *testing.T) {
	var vCardColumns = []string{"vcard"}

	s, mock := newVCardMock()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(vCardColumns).AddRow("<vCard><FN>Miguel Ángel</FN></vCard>"))

	vCard, err := s.FetchVCard(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, vCard)

	s, mock = newVCardMock()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(vCardColumns))

	vCard, err = s.FetchVCard(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Nil(t, vCard)

	s, mock = newVCardMock()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	vCard, _ = s.FetchVCard(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, vCard)
}

func newVCardMock() (*mySQLVCard, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLVCard{
		mySQLStorage: s,
	}, sqlMock
}
