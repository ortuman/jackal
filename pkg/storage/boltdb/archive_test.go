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

package boltdb

import (
	"context"
	"testing"
	"time"

	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBoltDB_InsertArchiveMessage(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBArchiveRep{tx: tx}

		m0 := testMessageStanza()

		err := rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
			ArchiveId: "a1234",
			Message:   m0.Proto(),
		})
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_FetchArchiveMetadata(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBArchiveRep{tx: tx}

		m0 := testMessageStanza()
		m1 := testMessageStanza()
		m2 := testMessageStanza()

		now0 := time.Now()
		now1 := now0.Add(time.Hour)
		now2 := now1.Add(time.Hour)

		err := rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
			ArchiveId: "a1234",
			Id:        "id0",
			Message:   m0.Proto(),
			Stamp:     timestamppb.New(now0),
		})
		require.NoError(t, err)

		err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
			ArchiveId: "a1234",
			Id:        "id1",
			Message:   m1.Proto(),
			Stamp:     timestamppb.New(now1),
		})
		require.NoError(t, err)

		err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
			ArchiveId: "a1234",
			Id:        "id2",
			Message:   m2.Proto(),
			Stamp:     timestamppb.New(now2),
		})
		require.NoError(t, err)

		metadata, err := rep.FetchArchiveMetadata(context.Background(), "a1234")
		require.NoError(t, err)

		require.Equal(t, "id0", metadata.StartId)
		require.Equal(t, now0.UTC().Format(archiveStampFormat), metadata.StartTimestamp)
		require.Equal(t, "id2", metadata.EndId)
		require.Equal(t, now2.UTC().Format(archiveStampFormat), metadata.EndTimestamp)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_DeleteArchive(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBArchiveRep{tx: tx}

		m0 := testMessageStanza()
		m1 := testMessageStanza()
		m2 := testMessageStanza()

		err := rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{ArchiveId: "a1234", Message: m0.Proto()})
		require.NoError(t, err)
		err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{ArchiveId: "a1234", Message: m1.Proto()})
		require.NoError(t, err)
		err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{ArchiveId: "a1234", Message: m2.Proto()})
		require.NoError(t, err)

		require.Equal(t, 3, countBucketElements(t, tx, archiveBucket("a1234")))

		require.NoError(t, rep.DeleteArchive(context.Background(), "a1234"))

		require.Equal(t, 0, countBucketElements(t, tx, archiveBucket("a1234")))

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_DeleteArchiveOldestMessages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBArchiveRep{tx: tx}

		m0 := testMessageStanza()
		m1 := testMessageStanza()
		m2 := testMessageStanza()

		err := rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
			ArchiveId: "a1234",
			Message:   m0.Proto(),
		})
		require.NoError(t, err)

		err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
			ArchiveId: "a1234",
			Message:   m1.Proto(),
		})
		require.NoError(t, err)

		err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
			ArchiveId: "a1234",
			Message:   m2.Proto(),
		})
		require.NoError(t, err)

		require.Equal(t, 3, countBucketElements(t, tx, archiveBucket("a1234")))

		err = rep.DeleteArchiveOldestMessages(context.Background(), "a1234", 2)
		require.NoError(t, err)

		require.Equal(t, 2, countBucketElements(t, tx, archiveBucket("a1234")))

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_FetchArchiveMessages(t *testing.T) {
	tcs := map[string]struct {
		filters           *archivemodel.Filters
		expectedResultIDs []string
	}{
		"filtering by jid": {
			filters: &archivemodel.Filters{
				With: "noelia@jackal.im",
			},
			expectedResultIDs: []string{"m0", "m1", "m3"},
		},
		"filtering by full jid": {
			filters: &archivemodel.Filters{
				With: "ortuman@jackal.im/firstwitch",
			},
			expectedResultIDs: []string{"m2"},
		},
		"filtering by ids": {
			filters: &archivemodel.Filters{
				Ids: []string{"m0", "m2"},
			},
			expectedResultIDs: []string{"m0", "m2"},
		},
		"filtering by after id": {
			filters: &archivemodel.Filters{
				AfterId: "m1",
			},
			expectedResultIDs: []string{"m2", "m3"},
		},
		"filtering by before id": {
			filters: &archivemodel.Filters{
				BeforeId: "m2",
			},
			expectedResultIDs: []string{"m0", "m1"},
		},
		"filtering by start": {
			filters: &archivemodel.Filters{
				Start: timestamppb.New(time.Date(2022, 01, 02, 00, 00, 00, 00, time.UTC)),
			},
			expectedResultIDs: []string{"m2", "m3"},
		},
		"filtering by end": {
			filters: &archivemodel.Filters{
				End: timestamppb.New(time.Date(2022, 01, 02, 00, 00, 00, 00, time.UTC)),
			},
			expectedResultIDs: []string{"m0"},
		},
	}
	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			db := setupDB(t)
			t.Cleanup(func() { cleanUp(db) })

			err := db.Update(func(tx *bolt.Tx) error {
				rep := boltDBArchiveRep{tx: tx}

				m0 := testMessageStanzaWithParameters("b0", "noelia@jackal.im/yard", "ortuman@jackal.im/chamber")
				m1 := testMessageStanzaWithParameters("b1", "noelia@jackal.im/orchard", "ortuman@jackal.im/balcony")
				m2 := testMessageStanzaWithParameters("b2", "witch1@jackal.im/yard", "ortuman@jackal.im/firstwitch")
				m3 := testMessageStanzaWithParameters("b3", "witch2@jackal.im/yard", "noelia@jackal.im/garden")

				err := rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
					ArchiveId: "a1234",
					Id:        "m0",
					FromJid:   "noelia@jackal.im/yard",
					ToJid:     "ortuman@jackal.im/chamber",
					Stamp:     timestamppb.New(time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)),
					Message:   m0.Proto(),
				})
				require.NoError(t, err)

				err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
					ArchiveId: "a1234",
					Id:        "m1",
					FromJid:   "noelia@jackal.im/orchard",
					ToJid:     "ortuman@jackal.im/balcony",
					Stamp:     timestamppb.New(time.Date(2022, 01, 02, 00, 00, 00, 00, time.UTC)),
					Message:   m1.Proto(),
				})
				require.NoError(t, err)

				err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
					ArchiveId: "a1234",
					Id:        "m2",
					FromJid:   "witch1@jackal.im/yard",
					ToJid:     "ortuman@jackal.im/firstwitch",
					Stamp:     timestamppb.New(time.Date(2022, 01, 03, 00, 00, 00, 00, time.UTC)),
					Message:   m2.Proto(),
				})
				require.NoError(t, err)

				err = rep.InsertArchiveMessage(context.Background(), &archivemodel.Message{
					ArchiveId: "a1234",
					Id:        "m3",
					FromJid:   "witch2@jackal.im/yard",
					ToJid:     "noelia@jackal.im/garden",
					Stamp:     timestamppb.New(time.Date(2022, 01, 04, 00, 00, 00, 00, time.UTC)),
					Message:   m3.Proto(),
				})
				require.NoError(t, err)

				messages, err := rep.FetchArchiveMessages(context.Background(), tc.filters, "a1234")
				require.NoError(t, err)

				var resultIDs []string
				for _, msg := range messages {
					resultIDs = append(resultIDs, msg.Id)
				}
				require.ElementsMatch(t, tc.expectedResultIDs, resultIDs)
				return nil
			})
			require.NoError(t, err)
		})
	}
}
