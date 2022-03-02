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
	"fmt"
	"os"
	"testing"

	"github.com/jackal-xmpp/stravaganza"

	bolt "go.etcd.io/bbolt"
)

func setupDB(t *testing.T) *bolt.DB {
	t.Helper()

	dbPath := fmt.Sprintf("%s/test.db", t.TempDir())
	db, err := bolt.Open(dbPath, 0666, nil)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func cleanUp(db *bolt.DB) {
	dbPath := db.Path()
	_ = db.Close()
	_ = os.RemoveAll(dbPath)
}

func testMessageStanza(body string) *stravaganza.Message {
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText(body).
			Build(),
	)
	msg, _ := b.BuildMessage()
	return msg
}
