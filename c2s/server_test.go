/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/util"
	"github.com/stretchr/testify/require"
)

func TestC2SSocketServer(t *testing.T) {
	host.Initialize([]host.Config{{Name: "localhost"}})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	router.Initialize(&router.Config{})

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
	go Initialize([]Config{cfg})

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

		Shutdown()
		errCh <- nil
	}()
	err := <-errCh
	require.Nil(t, err)

	router.Shutdown()
	storage.Shutdown()
	host.Shutdown()
}

func TestC2SWebSocketServer(t *testing.T) {
	privKeyFile := "../testdata/cert/test.server.key"
	certFile := "../testdata/cert/test.server.crt"
	cer, err := util.LoadCertificate(privKeyFile, certFile, "localhost")
	require.Nil(t, err)

	host.Initialize([]host.Config{{Name: "localhost", Certificate: cer}})
	router.Initialize(&router.Config{})
	storage.Initialize(&storage.Config{Type: storage.Memory})

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
	go Initialize([]Config{cfg})

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

		Shutdown()
		errCh <- nil
	}()
	err = <-errCh
	require.Nil(t, err)

	router.Shutdown()
	storage.Shutdown()
	host.Shutdown()
}
