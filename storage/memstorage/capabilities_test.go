package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertCapabilities(t *testing.T) {
	caps := model.Capabilities{Features: []string{"ns"}}
	s := New()
	s.EnableMockedError()
	err := s.InsertCapabilities("n1", "1234A", &caps)
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	err = s.InsertCapabilities("n1", "1234A", &caps)
	require.Nil(t, err)
}

func TestMemoryStorage_HasCapabilities(t *testing.T) {
	s := New()
	s.EnableMockedError()
	_, err := s.HasCapabilities("n1", "1234A")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	ok, err := s.HasCapabilities("n1", "1234A")
	require.Nil(t, err)
	require.False(t, ok)
}

func TestMemoryStorage_FetchCapabilities(t *testing.T) {
	caps := model.Capabilities{Features: []string{"ns"}}
	s := New()
	_ = s.InsertCapabilities("n1", "1234A", &caps)

	s.EnableMockedError()
	_, err := s.FetchCapabilities("n1", "1234A")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	cs, _ := s.FetchCapabilities("n1", "1234B")
	require.Nil(t, cs)

	cs, _ = s.FetchCapabilities("n1", "1234A")
	require.NotNil(t, cs)
}
