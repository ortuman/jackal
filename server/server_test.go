/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/transport"
	"github.com/stretchr/testify/require"
)

func TestSocketServer(t *testing.T) {
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	go func() {
		time.Sleep(time.Millisecond * 150)

		// test XMPP port...
		conn, err := net.Dial("tcp", "localhost:5123")
		require.Nil(t, err)
		require.NotNil(t, conn)

		xmlHdr := []byte(`<?xml version="1.0" encoding="UTF-8">`)
		n, err := conn.Write(xmlHdr)
		require.Nil(t, err)
		require.Equal(t, len(xmlHdr), n)
		conn.Close()

		time.Sleep(time.Millisecond * 150) // wait until disconnected

		// test debug port...
		req, err := http.NewRequest("GET", "http://localhost:9123/debug/pprof", nil)
		require.Nil(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		Shutdown()
	}()
	cfg := Config{
		ID: "srv-1234",
		TLS: TLSConfig{
			PrivKeyFile: "../testdata/cert/test.server.key",
			CertFile:    "../testdata/cert/test.server.crt",
		},
		Transport: TransportConfig{
			Type: transport.Socket,
			Port: 5123,
		},
	}
	Initialize([]Config{cfg}, 9123)
}

func TestWebSocketServer(t *testing.T) {
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	go func() {
		time.Sleep(time.Millisecond * 150)
		d := &websocket.Dialer{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		h := http.Header{"Sec-WebSocket-Protocol": []string{"xmpp"}}
		conn, _, err := d.Dial("wss://localhost:9876/srv-1234/ws", h)
		require.Nil(t, err)

		open := []byte(`<?xml version="1.0" encoding="UTF-8">`)
		err = conn.WriteMessage(websocket.TextMessage, open)
		require.Nil(t, err)
		conn.Close()

		time.Sleep(time.Millisecond * 150) // wait until disconnected

		Shutdown()
	}()
	cfg := Config{
		ID: "srv-1234",
		TLS: TLSConfig{
			PrivKeyFile: "../testdata/cert/test.server.key",
			CertFile:    "../testdata/cert/test.server.crt",
		},
		Transport: TransportConfig{
			Type: transport.WebSocket,
			Port: 9876,
		},
	}
	Initialize([]Config{cfg}, 0)
}
