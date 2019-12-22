/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"encoding/json"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestPgSQLInsertCapabilities(t *testing.T) {
	features := []string{"jabber:iq:last"}

	b, _ := json.Marshal(&features)

	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+)").
		WithArgs("n1", "1234A", b).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.InsertCapabilities(&model.Capabilities{Node: "n1", Ver: "1234A", Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)

	// error case
	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+)").
		WithArgs("n1", "1234A", b).
		WillReturnError(errGeneric)

	err = s.InsertCapabilities(&model.Capabilities{Node: "n1", Ver: "1234A", Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errGeneric, err)
}

func TestPgSQLFetchCapabilities(t *testing.T) {
	s, mock := NewMock()
	rows := sqlmock.NewRows([]string{"features"})
	rows.AddRow(`["jabber:iq:last"]`)

	mock.ExpectQuery("SELECT features FROM capabilities WHERE \\(node = . AND ver = .\\)").
		WithArgs("n1", "1234A").
		WillReturnRows(rows)

	caps, err := s.FetchCapabilities("n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.Equal(t, 1, len(caps.Features))
	require.Equal(t, "jabber:iq:last", caps.Features[0])

	// error case
	s, mock = NewMock()
	mock.ExpectQuery("SELECT features FROM capabilities WHERE \\(node = . AND ver = .\\)").
		WithArgs("n1", "1234A").
		WillReturnError(errGeneric)

	caps, err = s.FetchCapabilities("n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Nil(t, caps)
}
