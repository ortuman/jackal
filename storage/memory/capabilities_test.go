/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertCapabilities(t *testing.T) {
	caps := model.Capabilities{Node: "n1", Ver: "1234A", Features: []string{"ns"}}
	s := NewCapabilities()
	EnableMockedError()
	err := s.UpsertCapabilities(context.Background(), &caps)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()
	err = s.UpsertCapabilities(context.Background(), &caps)
	require.Nil(t, err)
}

func TestMemoryStorage_FetchCapabilities(t *testing.T) {
	caps := model.Capabilities{Node: "n1", Ver: "1234A", Features: []string{"ns"}}
	s := NewCapabilities()
	_ = s.UpsertCapabilities(context.Background(), &caps)

	EnableMockedError()
	_, err := s.FetchCapabilities(context.Background(), "n1", "1234A")
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	cs, _ := s.FetchCapabilities(context.Background(), "n1", "1234B")
	require.Nil(t, cs)

	cs, _ = s.FetchCapabilities(context.Background(), "n1", "1234A")
	require.NotNil(t, cs)
}
