package mysql

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMySQLInsertCapabilities(t *testing.T) {
	features := []string{"jabber:iq:last"}

	b, _ := json.Marshal(&features)

	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+)").
		WithArgs("n1", "1234A", b).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.InsertCapabilities("n1", "1234A", &model.Capabilities{Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)

	// error case
	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+)").
		WithArgs("n1", "1234A", b).
		WillReturnError(errMySQLStorage)

	err = s.InsertCapabilities("n1", "1234A", &model.Capabilities{Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLHasCapabilities(t *testing.T) {
	s, mock := NewMock()
	rows := sqlmock.NewRows([]string{"COUNT(*)"})
	rows.AddRow(1)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM capabilities WHERE \\(node = . AND ver = .\\)").
		WithArgs("n1", "1234A").
		WillReturnRows(rows)

	ok, err := s.HasCapabilities("n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.True(t, ok)

	// error case
	s, mock = NewMock()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM capabilities WHERE \\(node = . AND ver = .\\)").
		WithArgs("n1", "1234A").
		WillReturnError(errMySQLStorage)

	ok, err = s.HasCapabilities("n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.False(t, ok)
}

func TestMySQLFetchCapabilities(t *testing.T) {
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
		WillReturnError(errMySQLStorage)

	caps, err = s.FetchCapabilities("n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Nil(t, caps)
}
