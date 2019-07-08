package pgsql

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/stretchr/testify/require"
)

type optionArg struct {
	name  string
	value string
}

var optionArgs = []optionArg{
	{"pubsub#presence_based_delivery", "false"},
}

func TestStorageInsertPubSubNode(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO pubsub_nodes (.+) ON CONFLICT (.+) DO NOTHING").
		WithArgs("host", "name").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT id FROM pubsub_nodes WHERE (.+)").
		WithArgs("host", "name").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

	mock.ExpectExec("DELETE FROM pubsub_node_options WHERE (.+)").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	for _, optArg := range optionArgs {
		mock.ExpectExec("INSERT INTO pubsub_node_options (.+) VALUES (.+)").
			WithArgs(optArg.name, optArg.value).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	node := pubsubmodel.Node{Host: "host", Name: "name", Options: pubsubmodel.Options{}}
	err := s.InsertOrUpdatePubSubNode(&node)

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
}

func TestStorageGetPubSubNode(t *testing.T) {
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
		WillReturnError(errGeneric)

	_, err := s.GetPubSubNode("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errGeneric, err)
}
