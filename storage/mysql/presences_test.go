package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

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

func newPresencesMock() (*mySQLPresences, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLPresences{
		mySQLStorage: s,
		pool:         pool.NewBufferPool(),
	}, sqlMock
}
