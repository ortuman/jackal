/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger"
	"github.com/google/uuid"
)

type bgDBT struct {
	db *badger.DB

	dataDir string
}

func newT() *bgDBT {
	t := &bgDBT{}
	dir, _ := ioutil.TempDir("", "")
	t.dataDir = dir + "/com.jackal.tests.badgerdb." + uuid.New().String()

	if err := os.MkdirAll(filepath.Dir(t.dataDir), os.ModePerm); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	opts := badger.DefaultOptions(t.dataDir)
	db, err := badger.Open(opts)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	t.db = db
	return t
}

func (t *bgDBT) teardown() {
	_ = t.db.Close()
	_ = os.RemoveAll(t.dataDir)
}
