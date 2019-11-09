package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertCapabilities(t *testing.T) {
	caps := model.Capabilities{Node: "n1", Ver: "1234A", Features: []string{"ns"}}
	s := New()
	s.EnableMockedError()
	err := s.InsertCapabilities(&caps)
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	err = s.InsertCapabilities(&caps)
	require.Nil(t, err)
}

func TestMemoryStorage_FetchCapabilities(t *testing.T) {
	caps := model.Capabilities{Node: "n1", Ver: "1234A", Features: []string{"ns"}}
	s := New()
	_ = s.InsertCapabilities(&caps)

	s.EnableMockedError()
	_, err := s.FetchCapabilities("n1", "1234A")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	cs, _ := s.FetchCapabilities("n1", "1234B")
	require.Nil(t, cs)

	cs, _ = s.FetchCapabilities("n1", "1234A")
	require.NotNil(t, cs)
}
