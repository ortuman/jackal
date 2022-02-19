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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza"
	bolt "go.etcd.io/bbolt"
)

type boltDBOfflineRep struct {
	tx *bolt.Tx
}

func newOfflineRep(tx *bolt.Tx) *boltDBOfflineRep {
	return &boltDBOfflineRep{tx: tx}
}

func (r *boltDBOfflineRep) InsertOfflineMessage(_ context.Context, message *stravaganza.Message, username string) error {
	op := insertSeqOp{
		tx:     r.tx,
		bucket: offlineBucket(username),
		obj:    message,
	}
	return op.do()
}

func (r *boltDBOfflineRep) CountOfflineMessages(_ context.Context, username string) (int, error) {
	op := countKeysOp{
		tx:     r.tx,
		bucket: offlineBucket(username),
	}
	return op.do()
}

func (r *boltDBOfflineRep) FetchOfflineMessages(_ context.Context, username string) ([]*stravaganza.Message, error) {
	var retVal []*stravaganza.Message

	op := iterKeysOp{
		tx:     r.tx,
		bucket: offlineBucket(username),
		iterFn: func(_, b []byte) error {
			var elem stravaganza.PBElement
			if err := proto.Unmarshal(b, &elem); err != nil {
				return err
			}
			msg, err := stravaganza.NewBuilderFromProto(&elem).BuildMessage()
			if err != nil {
				return err
			}
			retVal = append(retVal, msg)
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func (r *boltDBOfflineRep) DeleteOfflineMessages(_ context.Context, username string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: offlineBucket(username),
	}
	return op.do()
}

func offlineBucket(username string) string {
	return fmt.Sprintf("offline:%s", username)
}

// InsertOfflineMessage satisfies repository.Offline interface.
func (r *Repository) InsertOfflineMessage(ctx context.Context, message *stravaganza.Message, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newOfflineRep(tx).InsertOfflineMessage(ctx, message, username)
	})
}

// CountOfflineMessages satisfies repository.Offline interface.
func (r *Repository) CountOfflineMessages(ctx context.Context, username string) (c int, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		c, err = newOfflineRep(tx).CountOfflineMessages(ctx, username)
		return err
	})
	return
}

// FetchOfflineMessages satisfies repository.Offline interface.
func (r *Repository) FetchOfflineMessages(ctx context.Context, username string) (msg []*stravaganza.Message, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		msg, err = newOfflineRep(tx).FetchOfflineMessages(ctx, username)
		return err
	})
	return
}

// DeleteOfflineMessages satisfies repository.Offline interface.
func (r *Repository) DeleteOfflineMessages(ctx context.Context, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newOfflineRep(tx).DeleteOfflineMessages(ctx, username)
	})
}
