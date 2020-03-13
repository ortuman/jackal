package mysql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	capsmodel "github.com/ortuman/jackal/model/capabilities"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestMySQLPresences_UpsertPresence(t *testing.T) {

	s, mock := newPresencesMock()
	mock.ExpectExec("INSERT INTO presences (.+) VALUES (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "jackal.im", "yard", `<presence from="ortuman@jackal.im/yard" to="ortuman@jackal.im"/>`, "", "", "alloc-1234", `<presence from="ortuman@jackal.im/yard" to="ortuman@jackal.im"/>`, "", "", "alloc-1234").
		WillReturnResult(sqlmock.NewResult(1, 1))

	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	inserted, err := s.UpsertPresence(context.Background(), xmpp.NewPresence(j, j.ToBareJID(), xmpp.AvailableType), j, "alloc-1234")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	require.True(t, inserted)
}

func TestMySQLPresences_FetchPresence(t *testing.T) {
	var columns = []string{"presence", "c.node", "c.ver", "c.features"}

	s, mock := newPresencesMock()
	mock.ExpectQuery("SELECT presence, c.node, c.ver, c.features FROM presences AS p, capabilities AS c WHERE \\(username = \\? AND domain = \\? AND resource = \\? AND p.node = c.node AND p.ver = c.ver\\)").
		WithArgs("ortuman", "jackal.im", "yard").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("<presence/>", "http://jackal.im", "v1234", `["urn:xmpp:ping"]`))

	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	presenceCaps, err := s.FetchPresence(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	require.NotNil(t, presenceCaps)

	require.Equal(t, "http://jackal.im", presenceCaps.Caps.Node)
	require.Equal(t, "v1234", presenceCaps.Caps.Ver)
	require.Len(t, presenceCaps.Caps.Features, 1)
	require.Equal(t, "urn:xmpp:ping", presenceCaps.Caps.Features[0])
}

func TestMySQLPresences_FetchPresencesMatchingJID(t *testing.T) {
	var columns = []string{"presence", "c.node", "c.ver", "c.features"}

	s, mock := newPresencesMock()
	mock.ExpectQuery("SELECT presence, c.node, c.ver, c.features FROM presences AS p, capabilities AS c WHERE \\(username = \\? AND domain = \\? AND resource = \\? AND p.node = c.node AND p.ver = c.ver\\)").
		WithArgs("ortuman", "jackal.im", "yard").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("<presence/>", "http://jackal.im", "v1234", `["urn:xmpp:ping"]`).
			AddRow("<presence/>", "http://jackal.im", "v1234", `["urn:xmpp:ping"]`),
		)

	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	presenceCaps, err := s.FetchPresencesMatchingJID(context.Background(), j)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	require.NotNil(t, presenceCaps)

	require.Equal(t, "http://jackal.im", presenceCaps[0].Caps.Node)
	require.Equal(t, "v1234", presenceCaps[0].Caps.Ver)
	require.Len(t, presenceCaps[0].Caps.Features, 1)
	require.Equal(t, "urn:xmpp:ping", presenceCaps[0].Caps.Features[0])
}

func TestMySQLPresences_DeletePresence(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)

	s, mock := newPresencesMock()
	mock.ExpectExec("DELETE FROM presences WHERE \\(username = \\? AND domain = \\? AND resource = \\?\\)").
		WithArgs(j.Node(), j.Domain(), j.Resource()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.DeletePresence(context.Background(), j)

	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestMySQLPresences_DeleteAllocationPresence(t *testing.T) {
	s, mock := newPresencesMock()
	mock.ExpectExec("DELETE FROM presences WHERE allocation_id = ?").
		WithArgs("alloc-1234").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.DeleteAllocationPresences(context.Background(), "alloc-1234")

	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestMySQLPresences_ClearPresences(t *testing.T) {
	s, mock := newPresencesMock()
	mock.ExpectExec("DELETE FROM presences").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.ClearPresences(context.Background())

	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestMySQLPresences_UpsertCapabilities(t *testing.T) {
	features := []string{"jabber:iq:last"}

	b, _ := json.Marshal(&features)

	s, mock := newPresencesMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+) ON DUPLICATE KEY UPDATE features = \\?, updated_at = NOW\\(\\)").
		WithArgs("n1", "1234A", b, b).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{Node: "n1", Ver: "1234A", Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)

	// error case
	s, mock = newPresencesMock()
	mock.ExpectExec("INSERT INTO capabilities (.+) VALUES (.+) ON DUPLICATE KEY UPDATE features = \\?, updated_at = NOW\\(\\)").
		WithArgs("n1", "1234A", b, b).
		WillReturnError(errMySQLStorage)

	err = s.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{Node: "n1", Ver: "1234A", Features: features})

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLPresences_FetchCapabilities(t *testing.T) {
	s, mock := newPresencesMock()
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
	s, mock = newPresencesMock()
	mock.ExpectQuery("SELECT features FROM capabilities WHERE \\(node = . AND ver = .\\)").
		WithArgs("n1", "1234A").
		WillReturnError(errMySQLStorage)

	caps, err = s.FetchCapabilities(context.Background(), "n1", "1234A")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Nil(t, caps)
}

func newPresencesMock() (*mySQLPresences, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLPresences{
		mySQLStorage: s,
		pool:         pool.NewBufferPool(),
	}, sqlMock
}
