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
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/stretchr/testify/require"
)

func TestSocketServer(t *testing.T) {
	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer Shutdown()

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

		// test debug port...
		req, err := http.NewRequest("GET", "http://localhost:9123/debug/pprof", nil)
		require.Nil(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		Shutdown()
	}()
	cfg := config.Server{
		ID: "srv-1234",
		TLS: config.TLS{
			PrivKeyFile: "../cert/key.pem",
			CertFile:    "../cert/cert.pem",
		},
		Transport: config.Transport{
			Type: config.SocketTransportType,
			Port: 5123,
		},
	}
	Initialize([]config.Server{cfg}, 9123)
}

func TestWebSocketServer(t *testing.T) {
	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer Shutdown()

	go func() {
		time.Sleep(time.Millisecond * 150)

		d := &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		h := http.Header{"Sec-WebSocket-Protocol": []string{"xmpp"}}
		conn, _, err := d.Dial("wss://localhost:9876/srv-1234/ws", h)
		require.Nil(t, err)

		open := []byte(`<?xml version="1.0" encoding="UTF-8">`)
		err = conn.WriteMessage(websocket.TextMessage, open)
		require.Nil(t, err)
		conn.Close()

		Shutdown()
	}()
	cfg := config.Server{
		ID: "srv-1234",
		TLS: config.TLS{
			PrivKeyFile: "../cert/key.pem",
			CertFile:    "../cert/cert.pem",
		},
		Transport: config.Transport{
			Type: config.WebSocketTransportType,
			Port: 9876,
		},
	}
	Initialize([]config.Server{cfg}, 0)
}
