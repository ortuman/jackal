// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgsqlrepository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/stretchr/testify/require"
)

func TestPgSQLPubSub_UpsertNode(t *testing.T) {
	// given
	opts := &pubsubmodel.Options{
		MaxItems:             100,
		DeliverNotifications: true,
	}
	optionsBytes, _ := proto.Marshal(opts)

	s, mock := newPubSubMock()
	mock.ExpectExec(`INSERT INTO pubsub_nodes \(host,name,options\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \(host, name\) DO UPDATE SET options = \$3`).
		WithArgs("ortuman@jackal.im", "princely_musings", optionsBytes).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: opts,
	})

	// then
	require.NoError(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLPubSub_FetchNode(t *testing.T) {
	// given
	opts := &pubsubmodel.Options{
		MaxItems:             100,
		DeliverNotifications: true,
	}
	optionsBytes, _ := proto.Marshal(opts)

	cols := []string{
		"id",
		"host",
		"name",
		"options",
	}

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT id, host, name, options FROM pubsub_nodes WHERE \(host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(int64(1), "ortuman@jackal.im", "princely_musings", optionsBytes),
		)

	// when
	node, err := s.FetchNode(context.Background(), "ortuman@jackal.im", "princely_musings")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, node)

	require.Equal(t, int64(1), node.Id)
	require.Equal(t, "ortuman@jackal.im", node.Host)
	require.Equal(t, "princely_musings", node.Name)
}

func TestPgSQLPubSub_FetchNodes(t *testing.T) {
	// given
	opts := &pubsubmodel.Options{
		MaxItems:             100,
		DeliverNotifications: true,
	}
	optionsBytes, _ := proto.Marshal(opts)

	cols := []string{
		"id",
		"host",
		"name",
		"options",
	}

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT id, host, name, options FROM pubsub_nodes WHERE host = \$1`).
		WithArgs("ortuman@jackal.im").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(int64(1), "ortuman@jackal.im", "princely_musings", optionsBytes),
		)

	// when
	nodes, err := s.FetchNodes(context.Background(), "ortuman@jackal.im")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Len(t, nodes, 1)

	require.Equal(t, int64(1), nodes[0].Id)
	require.Equal(t, "ortuman@jackal.im", nodes[0].Host)
	require.Equal(t, "princely_musings", nodes[0].Name)
}

func TestPgSQLPubSub_NodeExists(t *testing.T) {
	countCols := []string{"count"}

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM pubsub_nodes WHERE \(host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(
			sqlmock.NewRows(countCols).AddRow(1),
		)

	ok, err := s.NodeExists(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)
}

func TestPgSQLPubSub_DeleteNode(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_nodes WHERE \(host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteNode(context.Background(), "ortuman@jackal.im", "princely_musings")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLPubSub_DeleteNodes(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_nodes WHERE host = \$1`).
		WithArgs("ortuman@jackal.im").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteNodes(context.Background(), "ortuman@jackal.im")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLPubSub_UpsertNodeAffiliation(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`INSERT INTO pubsub_affiliations \(node_id,jid,affiliation\) VALUES \(\(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\),\$3,\$4\) ON CONFLICT \(node_id, jid\) DO UPDATE SET affiliation = \$4`).
		WithArgs("ortuman@jackal.im", "princely_musings", "ortuman@jackal.im", "owner").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		Jid:   "ortuman@jackal.im",
		State: pubsubmodel.AffiliationState_AFF_OWNER,
	}, "ortuman@jackal.im", "princely_musings")

	// then
	require.NoError(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLPubSub_FetchNodeAffiliation(t *testing.T) {
	cols := []string{
		"node_id",
		"jid",
		"affiliation",
	}

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT node_id, jid, affiliation FROM pubsub_affiliations WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\) AND jid = \$3`).
		WithArgs("ortuman@jackal.im", "princely_musings", "ortuman@jackal.im").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(int64(1), "ortuman@jackal.im", "owner"),
		)

	// when
	aff, err := s.FetchNodeAffiliation(context.Background(), "ortuman@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.NoError(t, err)
	require.NotNil(t, aff)

	require.Equal(t, int64(1), aff.NodeId)
	require.Equal(t, "ortuman@jackal.im", aff.Jid)
	require.Equal(t, pubsubmodel.AffiliationState_AFF_OWNER, aff.State)
}

func TestPgSQLPubSub_FetchNodeAffiliations(t *testing.T) {
	cols := []string{
		"node_id",
		"jid",
		"affiliation",
	}

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT node_id, jid, affiliation FROM pubsub_affiliations WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(int64(1), "ortuman@jackal.im", "owner"),
		)

	// when
	affiliations, err := s.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.NoError(t, err)
	require.Len(t, affiliations, 1)

	require.Equal(t, int64(1), affiliations[0].NodeId)
	require.Equal(t, "ortuman@jackal.im", affiliations[0].Jid)
	require.Equal(t, pubsubmodel.AffiliationState_AFF_OWNER, affiliations[0].State)
}

func TestPgSQLPubSub_DeleteNodeAffiliation(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_affiliations WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\) AND jid = \$3`).
		WithArgs("ortuman@jackal.im", "princely_musings", "ortuman@jackal.im").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteNodeAffiliation(context.Background(), "ortuman@jackal.im", "ortuman@jackal.im", "princely_musings")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLPubSub_DeleteNodeAffiliations(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_affiliations WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLPubSub_UpsertNodeSubscription(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`INSERT INTO pubsub_subscriptions \(node_id,id,jid,subscription\) VALUES \(\(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\),\$3,\$4\,\$5\) ON CONFLICT \(node_id, jid\) DO UPDATE SET subscription = \$5`).
		WithArgs("ortuman@jackal.im", "princely_musings", "1234", "ortuman@jackal.im", "subscribed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		Id:    "1234",
		Jid:   "ortuman@jackal.im",
		State: pubsubmodel.SubscriptionState_SUB_SUBSCRIBED,
	}, "ortuman@jackal.im", "princely_musings")

	// then
	require.NoError(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLPubSub_FetchNodeSubscription(t *testing.T) {
	cols := []string{
		"node_id",
		"id",
		"jid",
		"subscription",
	}

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT node_id, id, jid, subscription FROM pubsub_subscriptions WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\) AND jid = \$3`).
		WithArgs("ortuman@jackal.im", "princely_musings", "ortuman@jackal.im").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(int64(1), "1234", "ortuman@jackal.im", "subscribed"),
		)

	// when
	sub, err := s.FetchNodeSubscription(context.Background(), "ortuman@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.NoError(t, err)
	require.NotNil(t, sub)

	require.Equal(t, int64(1), sub.NodeId)
	require.Equal(t, "1234", sub.Id)
	require.Equal(t, "ortuman@jackal.im", sub.Jid)
	require.Equal(t, pubsubmodel.SubscriptionState_SUB_SUBSCRIBED, sub.State)
}

func TestPgSQLPubSub_FetchNodeSubscriptions(t *testing.T) {
	cols := []string{
		"node_id",
		"id",
		"jid",
		"subscription",
	}

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT node_id, id, jid, subscription FROM pubsub_subscriptions WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(int64(1), "1234", "ortuman@jackal.im", "subscribed"),
		)

	// when
	subs, err := s.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.NoError(t, err)
	require.Len(t, subs, 1)

	require.Equal(t, int64(1), subs[0].NodeId)
	require.Equal(t, "1234", subs[0].Id)
	require.Equal(t, "ortuman@jackal.im", subs[0].Jid)
	require.Equal(t, pubsubmodel.SubscriptionState_SUB_SUBSCRIBED, subs[0].State)
}

func TestPgSQLPubSub_DeleteNodeSubscription(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_subscriptions WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\) AND jid = \$3`).
		WithArgs("ortuman@jackal.im", "princely_musings", "ortuman@jackal.im").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteNodeSubscription(context.Background(), "ortuman@jackal.im", "ortuman@jackal.im", "princely_musings")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLPubSub_DeleteNodeSubscriptions(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_subscriptions WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLPubSub_InsertNodeItem(t *testing.T) {
	// given
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	expectedPayload, _ := proto.Marshal(msg.Proto())

	s, mock := newPubSubMock()
	mock.ExpectExec(`INSERT INTO pubsub_items \(node_id,id,publisher,payload\) VALUES \(\(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\),\$3,\$4\,\$5\)`).
		WithArgs("ortuman@jackal.im", "princely_musings", "1234", "ortuman@jackal.im", expectedPayload).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.InsertNodeItem(context.Background(), &pubsubmodel.Item{
		Id:        "1234",
		Publisher: "ortuman@jackal.im",
		Payload:   msg.Proto(),
	}, "ortuman@jackal.im", "princely_musings")

	// then
	require.NoError(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLPubSub_FetchNodeItems(t *testing.T) {
	cols := []string{
		"node_id",
		"id",
		"publisher",
		"payload",
	}
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	expectedPayload, _ := proto.Marshal(msg.Proto())

	s, mock := newPubSubMock()
	mock.ExpectQuery(`SELECT node_id, id, publisher, payload FROM pubsub_items WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(int64(1), "1234", "ortuman@jackal.im", expectedPayload),
		)

	// when
	items, err := s.FetchNodeItems(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.NoError(t, err)
	require.Len(t, items, 1)

	require.Equal(t, int64(1), items[0].NodeId)
	require.Equal(t, "1234", items[0].Id)
	require.Equal(t, "ortuman@jackal.im", items[0].Publisher)
	require.Equal(t, stravaganza.MessageName, items[0].Payload.Name)
}

func TestPgSQLPubSub_DeleteOldestNodeItems(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_items WHERE \(node_id = \(SELECT "id" FROM pubsub_nodes WHERE host = \$1 AND name = \$2\) AND "id" NOT IN \(SELECT "id" FROM pubsub_items WHERE host = \$3 AND name = \$4 ORDER BY created_at DESC LIMIT \$5 OFFSET 0\)\)`).
		WithArgs("ortuman@jackal.im", "princely_musings", "ortuman@jackal.im", "princely_musings", 10).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteOldestNodeItems(context.Background(), "ortuman@jackal.im", "princely_musings", 10)

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLPubSub_DeleteNodeItems(t *testing.T) {
	// given
	s, mock := newPubSubMock()
	mock.ExpectExec(`DELETE FROM pubsub_items WHERE node_id = \(SELECT id FROM pubsub_nodes WHERE host = \$1 AND name = \$2\)`).
		WithArgs("ortuman@jackal.im", "princely_musings").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteNodeItems(context.Background(), "ortuman@jackal.im", "princely_musings")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func newPubSubMock() (*pgSQLPubSubRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLPubSubRep{conn: s}, sqlMock
}
