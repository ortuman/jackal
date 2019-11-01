package mysql

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestMySQLUpsertPubSubNode(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO pubsub_nodes (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("host", "name").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT id FROM pubsub_nodes WHERE (.+)").
		WithArgs("host", "name").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

	mock.ExpectExec("DELETE FROM pubsub_node_options WHERE (.+)").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	opts := pubsubmodel.Options{}

	optMap, _ := opts.Map()
	for i := 0; i < len(optMap); i++ {
		mock.ExpectExec("INSERT INTO pubsub_node_options (.+)").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	node := pubsubmodel.Node{Host: "host", Name: "name", Options: opts}
	err := s.UpsertPubSubNode(&node)

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
}

func TestMySQLFetchPubSubNode(t *testing.T) {
	var cols = []string{"name", "value"}

	s, mock := NewMock()
	rows := sqlmock.NewRows(cols)
	rows.AddRow("pubsub#access_model", "presence")
	rows.AddRow("pubsub#publish_model", "publishers")
	rows.AddRow("pubsub#send_last_published_item", "on_sub_and_presence")

	mock.ExpectQuery("SELECT name, value FROM pubsub_node_options WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(rows)

	node, err := s.FetchPubSubNode("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.NotNil(t, node)
	require.Equal(t, node.Options.AccessModel, pubsubmodel.Presence)
	require.Equal(t, node.Options.PublishModel, pubsubmodel.Publishers)
	require.Equal(t, node.Options.SendLastPublishedItem, pubsubmodel.OnSubAndPresence)

	// error case
	s, mock = NewMock()
	mock.ExpectQuery("SELECT name, value FROM pubsub_node_options WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchPubSubNode("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLDeletePubSubNode(t *testing.T) {
	s, mock := NewMock()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM pubsub_nodes WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

	mock.ExpectExec("DELETE FROM pubsub_nodes WHERE (.+)").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM pubsub_node_options WHERE (.+)").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM pubsub_items WHERE (.+)").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM pubsub_affiliations WHERE (.+)").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM pubsub_subscriptions WHERE (.+)").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeletePubSubNode("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestMySQLUpsertPubSubNodeItem(t *testing.T) {
	payload := xmpp.NewIQType(uuid.New().String(), xmpp.GetType)

	s, mock := NewMock()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM pubsub_nodes WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

	mock.ExpectExec("INSERT INTO pubsub_items (.+) ON DUPLICATE KEY UPDATE payload = (.+), publisher = (.+), updated_at = NOW()").
		WithArgs("1", "abc1234", payload.String(), "ortuman@jackal.im", payload.String(), "ortuman@jackal.im").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT COUNT(.+) FROM pubsub_items WHERE node_id = (.+)").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow("1"))

	mock.ExpectQuery("SELECT MIN(.+) FROM pubsub_items WHERE node_id = (.+)").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"MIN(created_at)"}).AddRow("2019-07-14 10:24:42"))

	mock.ExpectExec("DELETE FROM pubsub_items WHERE (.+)").
		WithArgs("1", "2019-07-14 10:24:42").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.UpsertPubSubNodeItem(&pubsubmodel.Item{
		ID:        "abc1234",
		Publisher: "ortuman@jackal.im",
		Payload:   payload,
	}, "ortuman@jackal.im", "princely_musings", 1)

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
}

func TestMySQLFetchPubSubNodeItems(t *testing.T) {
	s, mock := NewMock()
	rows := sqlmock.NewRows([]string{"item_id", "publisher", "payload"})
	rows.AddRow("1234", "ortuman@jackal.im", "<message/>")
	rows.AddRow("5678", "noelia@jackal.im", "<iq type='get'/>")

	mock.ExpectQuery("SELECT item_id, publisher, payload FROM pubsub_items WHERE node_id = (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(rows)

	items, err := s.FetchPubSubNodeItems("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.Equal(t, 2, len(items))
	require.Equal(t, "1234", items[0].ID)
	require.Equal(t, "5678", items[1].ID)

	// error case
	s, mock = NewMock()
	mock.ExpectQuery("SELECT item_id, publisher, payload FROM pubsub_items WHERE node_id = (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchPubSubNodeItems("ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLFetchPubSubNodeItemsWithID(t *testing.T) {
	s, mock := NewMock()
	rows := sqlmock.NewRows([]string{"item_id", "publisher", "payload"})
	rows.AddRow("1234", "ortuman@jackal.im", "<message/>")
	rows.AddRow("5678", "noelia@jackal.im", "<iq type='get'/>")

	identifiers := []string{"1234", "5678"}

	mock.ExpectQuery("SELECT item_id, publisher, payload FROM pubsub_items WHERE (.+ IN (.+)) ORDER BY created_at").
		WithArgs("ortuman@jackal.im", "princely_musings", "1234", "5678").
		WillReturnRows(rows)

	items, err := s.FetchPubSubNodeItemsWithIDs("ortuman@jackal.im", "princely_musings", identifiers)

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.Equal(t, 2, len(items))
	require.Equal(t, "1234", items[0].ID)
	require.Equal(t, "5678", items[1].ID)

	// error case
	s, mock = NewMock()
	mock.ExpectQuery("SELECT item_id, publisher, payload FROM pubsub_items WHERE (.+ IN (.+)) ORDER BY created_at").
		WithArgs("ortuman@jackal.im", "princely_musings", "1234", "5678").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchPubSubNodeItemsWithIDs("ortuman@jackal.im", "princely_musings", identifiers)

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLUpsertPubSubNodeAffiliation(t *testing.T) {
	s, mock := NewMock()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM pubsub_nodes WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

	mock.ExpectExec("INSERT INTO pubsub_affiliations (.+) VALUES (.+) ON DUPLICATE KEY UPDATE affiliation = (.+), updated_at = (.+)").
		WithArgs("1", "ortuman@jackal.im", "owner", "owner").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: "owner",
	}, "ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
}

func TestMySQLFetchPubSubNodeAffiliations(t *testing.T) {
	s, mock := NewMock()
	rows := sqlmock.NewRows([]string{"jid", "affiliation"})
	rows.AddRow("ortuman@jackal.im", "owner")
	rows.AddRow("noelia@jackal.im", "publisher")

	mock.ExpectQuery("SELECT jid, affiliation FROM pubsub_affiliations WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(rows)

	affiliations, err := s.FetchPubSubNodeAffiliations("ortuman@jackal.im", "princely_musings")
	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.Equal(t, 2, len(affiliations))

	// error case
	mock.ExpectQuery("SELECT jid, affiliation FROM pubsub_affiliations WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnError(errMySQLStorage)

	affiliations, err = s.FetchPubSubNodeAffiliations("ortuman@jackal.im", "princely_musings")
	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestPgSQLDeletePubSubNodeAffiliation(t *testing.T) {
	s, mock := NewMock()

	mock.ExpectExec("DELETE FROM pubsub_affiliations WHERE (.+)").
		WithArgs("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeletePubSubNodeAffiliation("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)

	// error case
	s, mock = NewMock()
	mock.ExpectExec("DELETE FROM pubsub_affiliations WHERE (.+)").
		WithArgs("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings").
		WillReturnError(errMySQLStorage)

	err = s.DeletePubSubNodeAffiliation("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLUpsertPubSubNodeSubscription(t *testing.T) {
	s, mock := NewMock()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM pubsub_nodes WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

	mock.ExpectExec("INSERT INTO pubsub_subscriptions (.+) VALUES (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("1", "1234", "ortuman@jackal.im", "subscribed", "1234", "subscribed").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		SubID:        "1234",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}, "ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
}

func TestMySQLFetchPubSubNodeSubscriptions(t *testing.T) {
	s, mock := NewMock()
	rows := sqlmock.NewRows([]string{"subid", "jid", "subscription"})
	rows.AddRow("1234", "ortuman@jackal.im", "subscribed")
	rows.AddRow("5678", "noelia@jackal.im", "unsubscribed")

	mock.ExpectQuery("SELECT subid, jid, subscription FROM pubsub_subscriptions WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(rows)

	subscriptions, err := s.FetchPubSubNodeSubscriptions("ortuman@jackal.im", "princely_musings")
	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)
	require.Equal(t, 2, len(subscriptions))

	// error case
	mock.ExpectQuery("SELECT subid, jid, subscription FROM pubsub_subscriptions WHERE (.+)").
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnError(errMySQLStorage)

	subscriptions, err = s.FetchPubSubNodeSubscriptions("ortuman@jackal.im", "princely_musings")
	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLDeletePubSubNodeSubscription(t *testing.T) {
	s, mock := NewMock()

	mock.ExpectExec("DELETE FROM pubsub_subscriptions WHERE (.+)").
		WithArgs("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeletePubSubNodeSubscription("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, err)

	// error case
	s, mock = NewMock()
	mock.ExpectExec("DELETE FROM pubsub_subscriptions WHERE (.+)").
		WithArgs("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings").
		WillReturnError(errMySQLStorage)

	err = s.DeletePubSubNodeSubscription("noeliac@jackal.im", "ortuman@jackal.im", "princely_musings")

	require.Nil(t, mock.ExpectationsWereMet())

	require.NotNil(t, err)
	require.Equal(t, errMySQLStorage, err)
}
