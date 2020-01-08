/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_Capabilities(t *testing.T) {
	t.Parallel()

	s, teardown := newCapabilitiesMock()
	defer teardown()

	caps := model.Capabilities{Node: "n1", Ver: "1234AB", Features: []string{"ns"}}

	err := s.UpsertCapabilities(context.Background(), &caps)
	require.Nil(t, err)

	cs, err := s.FetchCapabilities(context.Background(), "n1", "1234AB")
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, "ns", cs.Features[0])

	cs2, err := s.FetchCapabilities(context.Background(), "n2", "1234AB")
	require.Nil(t, cs2)
	require.Nil(t, err)
}

func newCapabilitiesMock() (*badgerDBCapabilities, func()) {
	t := newT()
	return &badgerDBCapabilities{badgerDBStorage: newStorage(t.db)}, func() {
		t.teardown()
	}
}
