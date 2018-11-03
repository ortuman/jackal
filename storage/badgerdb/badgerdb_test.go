/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"io/ioutil"
	"os"

	"github.com/pborman/uuid"
)

type testBadgerDBHelper struct {
	db      *Storage
	dataDir string
}

func tUtilBadgerDBSetup() *testBadgerDBHelper {
	h := &testBadgerDBHelper{}
	dir, _ := ioutil.TempDir("", "")
	h.dataDir = dir + "/com.jackal.tests.badgerdb." + uuid.New()
	cfg := Config{DataDir: h.dataDir}
	h.db = New(&cfg)
	return h
}

func tUtilBadgerDBTeardown(h *testBadgerDBHelper) {
	h.db.Close()
	os.RemoveAll(h.dataDir)
}
