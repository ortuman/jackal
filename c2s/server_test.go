/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"testing"
	"time"

	c2srouter "github.com/ortuman/jackal/c2s/router"

	"github.com/ortuman/jackal/stream"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/transport"
	utiltls "github.com/ortuman/jackal/util/tls"
	"github.com/stretchr/testify/require"
)

func TestC2SSocketServer(t *testing.T) {
	r, _, _ := setupTest("localhost")

	errCh := make(chan error)
	cfg := Config{
		ID:               "srv-1234",
		ConnectTimeout:   time.Second * time.Duration(5),
		MaxStanzaSize:    8192,
		ResourceConflict: Reject,
		Transport: TransportConfig{
			Type: transport.Socket,
			Port: 9998,
		},
	}
	srv := server{
		cfg:           &cfg,
		router:        r,
		mods:          &module.Modules{},
		comps:         &component.Components{},
		inConnections: make(map[string]stream.C2S),
	}
	go srv.start()

	go func() {
		time.Sleep(time.Millisecond * 150)

		// test XMPP port...
		conn, err := net.Dial("tcp", "127.0.0.1:9998")
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

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		_ = srv.shutdown(ctx)
		errCh <- nil
	}()
	err := <-errCh
	require.Nil(t, err)
}

func TestC2SWebSocketServer(t *testing.T) {
	privKeyFile := "../testdata/cert/test.server.key"
	certFile := "../testdata/cert/test.server.crt"
	cer, err := utiltls.LoadCertificate(privKeyFile, certFile, "localhost")
	require.Nil(t, err)

	r, _ := router.New(
		&router.Config{
			Hosts: []router.HostConfig{{Name: "localhost", Certificate: cer}},
		},
		c2srouter.New(memorystorage.NewUser()),
		memorystorage.NewBlockList(),
	)
	errCh := make(chan error)
	cfg := Config{
		ID:               "srv-1234",
		ConnectTimeout:   time.Second * time.Duration(5),
		MaxStanzaSize:    8192,
		ResourceConflict: Reject,
		Transport: TransportConfig{
			Type:    transport.WebSocket,
			URLPath: "/xmpp/ws",
			Port:    9999,
		},
	}
	srv := server{
		cfg:           &cfg,
		router:        r,
		mods:          &module.Modules{},
		comps:         &component.Components{},
		inConnections: make(map[string]stream.C2S),
	}
	go srv.start()

	go func() {
		time.Sleep(time.Millisecond * 150)
		d := &websocket.Dialer{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		h := http.Header{"Sec-WebSocket-Protocol": []string{"xmpp"}}
		conn, _, err := d.Dial("wss://127.0.0.1:9999/xmpp/ws", h)
		if err != nil {
			errCh <- err
			return
		}
		open := []byte(`<?xml version="1.0" encoding="UTF-8">`)
		err = conn.WriteMessage(websocket.TextMessage, open)
		if err != nil {
			errCh <- err
			return
		}

		time.Sleep(time.Millisecond * 150) // wait until disconnected

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		_ = srv.shutdown(ctx)
		errCh <- nil
	}()
	err = <-errCh
	require.Nil(t, err)
}
