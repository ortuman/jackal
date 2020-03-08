/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"encoding/json"
	"testing"

	capsmodel "github.com/ortuman/jackal/model/capabilities"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMySQLUpsertCapabilities(t *testing.T) {
	features := []string{"jabber:iq:last"}

	b, _ := json.Marshal(&features)

	s, mock := newCapabilitiesMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+) ON DUPLICATE KEY UPDATE features = \\?, updated_at = NOW\\(\\)").
		WithArgs("n1", "1234A", b, b).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{Node: "n1", Ver: "1234A", Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)

	// error case
	s, mock = newCapabilitiesMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+) ON DUPLICATE KEY UPDATE features = \\?, updated_at = NOW\\(\\)").
		WithArgs("n1", "1234A", b, b).
		WillReturnError(errMySQLStorage)

	err = s.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{Node: "n1", Ver: "1234A", Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLFetchCapabilities(t *testing.T) {
	s, mock := newCapabilitiesMock()
	rows := sqlmock.NewRows([]string{"features"})
	rows.AddRow(`["jabber:iq:last"]`)

	mock.ExpectQuery("SELECT features FROM capabilities WHERE \\(node = . AND ver = .\\)").
		WithArgs("n1", "1234A").
		WillReturnRows(rows)

	caps, err := s.FetchCapabilities(context.Background(), "n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.Equal(t, 1, len(caps.Features))
	require.Equal(t, "jabber:iq:last", caps.Features[0])

	// error case
	s, mock = newCapabilitiesMock()
	mock.ExpectQuery("SELECT features FROM capabilities WHERE \\(node = . AND ver = .\\)").
		WithArgs("n1", "1234A").
		WillReturnError(errMySQLStorage)

	caps, err = s.FetchCapabilities(context.Background(), "n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Nil(t, caps)
}

func newCapabilitiesMock() (*mySQLCapabilities, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLCapabilities{
		mySQLStorage: s,
	}, sqlMock
}
