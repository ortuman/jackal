/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"testing"
	"time"

	c2srouter "github.com/ortuman/jackal/c2s/router"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
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
	defer func() { _ = mods.Shutdown(context.Background()) }()

	require.Equal(t, 10, len(mods.all))
}

func TestModules_ProcessIQ(t *testing.T) {
	mods := setupModules(t)
	defer func() { _ = mods.Shutdown(context.Background()) }()

	j0, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	j1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)

	stm := stream.NewMockC2S(uuid.New().String(), j0)
	mods.router.Bind(context.Background(), stm)

	iqID := uuid.New().String()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j0)
	iq.SetToJID(j1)
	mods.ProcessIQ(context.Background(), iq)

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

	rep, _ := storage.New(&storage.Config{Type: storage.Memory})
	r, _ := router.New(
		&router.Config{
			Hosts: []router.HostConfig{{Name: "jackal.im", Certificate: tls.Certificate{}}},
		},
		c2srouter.New(rep.User(), rep.BlockList()),
	)
	return New(&config, r, rep)
}
