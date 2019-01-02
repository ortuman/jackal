/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

type fakeModule struct {
	shutdownCh chan bool
}

func (m *fakeModule) Shutdown() error {
	if m.shutdownCh != nil {
		close(m.shutdownCh)
	}
	return nil
}

func TestModules_New(t *testing.T) {
	mods := setupModules(t)
	defer mods.Shutdown(context.Background())

	require.Equal(t, 10, len(mods.all))
}

func TestModules_ProcessIQ(t *testing.T) {
	mods := setupModules(t)
	defer mods.Shutdown(context.Background())

	j0, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	j1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	stm := stream.NewMockC2S(uuid.New().String(), j1)

	iqID := uuid.New().String()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j0)
	iq.SetToJID(j1)
	mods.ProcessIQ(iq, stm)

	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.IQName, elem.Name())
	require.Equal(t, xmpp.ErrorType, elem.Type())
}

func TestModules_Shutdown(t *testing.T) {
	mods := setupModules(t)

	var mod fakeModule
	mod.shutdownCh = make(chan bool)

	mods.all = append(mods.all, &mod)
	_ = mods.Shutdown(context.Background())

	select {
	case <-mod.shutdownCh:
		break
	case <-time.After(time.Millisecond * 250):
		require.Fail(t, "modules shutdown timeout")
	}
}

func setupModules(t *testing.T) *Modules {
	var config Config
	b, err := ioutil.ReadFile("../testdata/config_modules.yml")
	require.Nil(t, err)
	err = yaml.Unmarshal(b, &config)
	require.Nil(t, err)

	return New(&config, &router.Router{})
}
