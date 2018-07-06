/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"net"
	"testing"
	"time"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/stretchr/testify/require"
)

func TestS2SSocketServer(t *testing.T) {
	host.Initialize([]host.Config{{Name: "localhost"}})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	router.Initialize(&router.Config{})

	errCh := make(chan error)
	cfg := Config{
		Enabled:        true,
		ConnectTimeout: time.Second * time.Duration(5),
		MaxStanzaSize:  8192,
		Transport: TransportConfig{
			Port:      12778,
			KeepAlive: time.Duration(600) * time.Second,
		},
	}
	go Initialize(&cfg, &module.Config{})

	go func() {
		time.Sleep(time.Millisecond * 150)

		// test XMPP port...
		conn, err := net.Dial("tcp", "127.0.0.1:12778")
		if err != nil {
			errCh <- err
			return
		}
		xmlHdr := []byte(`<?xml version="1.0" encoding="UTF-8">`)
		_, err = conn.Write(xmlHdr)
		if err != nil {
			errCh <- err
			return
		}

		time.Sleep(time.Millisecond * 150) // wait until disconnected

		Shutdown()
		errCh <- nil
	}()
	err := <-errCh
	require.Nil(t, err)

	router.Shutdown()
	storage.Shutdown()
	host.Shutdown()
}
