package mysql

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertPubSubNode(t *testing.T) {
}

func TestMySQLStorageGetPubSubNode(t *testing.T) {
	var cols = []string{"name", "value"}

	s, mock := NewMock()
	rows := sqlmock.NewRows(cols)
	rows.AddRow("pubsub#access_model", "presence")
	rows.AddRow("pubsub#publish_model", "publishers")
	rows.AddRow("pubsub#send_last_published_item", "on_sub_and_presence")

	mock.ExpectQuery("SELECT name, value FROM pubsub_node_options WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(rows)

	node, err := s.GetPubSubNode("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.NotNil(t, node)
	require.Equal(t, node.Options.AccessModel, pubsubmodel.Presence)
	require.Equal(t, node.Options.PublishModel, pubsubmodel.Publishers)
	require.Equal(t, node.Options.SendLastPublishedItem, pubsubmodel.OnSubAndPresence)
}

func TestMySQLStorageGetPubSubNodeError(t *testing.T) {

	s, mock := NewMock()
	mock.ExpectQuery("SELECT name, value FROM pubsub_node_options WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnError(errMySQLStorage)

	_, err := s.GetPubSubNode("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}
