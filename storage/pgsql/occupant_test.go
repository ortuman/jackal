/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestPgSQLStorageInsertOccupant(t *testing.T) {
	j, _ := jid.NewWithString("room@conference.jackal.im/nick", true)
	o, _ := mucmodel.NewOccupant(j, j.ToBareJID())
	o.AddResource("yard")
	o.SetAffiliation("owner")
	o.SetRole("moderator")

	s, mock := newOccupantMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO occupants (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(o.OccupantJID.String(), o.BareJID.String(), o.GetAffiliation(), o.GetRole()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO resources (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(o.OccupantJID.String(), "yard").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := s.UpsertOccupant(context.Background(), o)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newOccupantMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO occupants (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(o.OccupantJID.String(), o.BareJID.String(), o.GetAffiliation(), o.GetRole()).
		WillReturnError(errMocked)
	mock.ExpectRollback()

	err = s.UpsertOccupant(context.Background(), o)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, err, errMocked)
}

func TestPgSQLStorageDeleteOccupant(t *testing.T) {
	j, _ := jid.NewWithString("room@conference.jackal.im/nick", true)
	s, mock := newOccupantMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM occupants (.+)").
		WithArgs(j.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM resources (.+)").
		WithArgs(j.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteOccupant(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newOccupantMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM occupants (.+)").
		WithArgs(j.String()).WillReturnError(errMocked)
	mock.ExpectRollback()

	err = s.DeleteOccupant(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func TestPgSQLStorageFetchOccupant(t *testing.T) {
	j, _ := jid.NewWithString("room@conference.jackal.im/nick", true)

	occColumns := []string{"occupant_jid", "bare_jid", "affiliation", "role"}
	resColumns := []string{"occupant_jid", "resource"}

	s, mock := newOccupantMock()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM occupants (.+)").
		WithArgs(j.String()).
		WillReturnRows(sqlmock.NewRows(occColumns))
	mock.ExpectCommit()

	occ, _ := s.FetchOccupant(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, occ)

	s, mock = newOccupantMock()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM occupants (.+)").
		WithArgs(j.String()).
		WillReturnRows(sqlmock.NewRows(occColumns).
			AddRow(j.String(), j.ToBareJID().String(), "owner", "moderator"))
	mock.ExpectQuery("SELECT (.+) FROM resources (.+)").
		WithArgs(j.String()).
		WillReturnRows(sqlmock.NewRows(resColumns).
			AddRow(j.String(), "phone"))
	mock.ExpectCommit()
	occ, err := s.FetchOccupant(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, occ)
	require.Equal(t, occ.OccupantJID.String(), j.String())
	require.Equal(t, occ.BareJID.String(), j.ToBareJID().String())
	require.Equal(t, occ.GetAffiliation(), "owner")
	require.Equal(t, occ.GetRole(), "moderator")
	require.Len(t, occ.GetAllResources(), 1)
	require.Equal(t, occ.GetAllResources()[0], "phone")

	s, mock = newOccupantMock()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM occupants (.+)").
		WithArgs(j.String()).WillReturnError(errMocked)
	mock.ExpectRollback()
	_, err = s.FetchOccupant(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func TestPgSQLStorageOccupantExists(t *testing.T) {
	j, _ := jid.NewWithString("room@conference.jackal.im/nick", true)
	countCols := []string{"count"}

	s, mock := newOccupantMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM occupants (.+)").
		WithArgs(j.String()).
		WillReturnRows(sqlmock.NewRows(countCols).AddRow(1))

	ok, err := s.OccupantExists(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)

	s, mock = newOccupantMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM occupants (.+)").
		WithArgs(j.String()).
		WillReturnError(errMocked)
	_, err = s.OccupantExists(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func newOccupantMock() (*pgSQLOccupant, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &pgSQLOccupant{
		pgSQLStorage: s,
	}, sqlMock
}
