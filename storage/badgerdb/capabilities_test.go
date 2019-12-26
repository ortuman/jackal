package badgerdb

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_Capabilities(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	caps := model.Capabilities{Node: "n1", Ver: "1234AB", Features: []string{"ns"}}

	err := h.db.InsertCapabilities(context.Background(), &caps)
	require.Nil(t, err)

	cs, err := h.db.FetchCapabilities(context.Background(), "n1", "1234AB")
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, "ns", cs.Features[0])

	cs2, err := h.db.FetchCapabilities(context.Background(), "n2", "1234AB")
	require.Nil(t, cs2)
	require.Nil(t, err)
}
