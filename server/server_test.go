/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer Shutdown()

	go func() {
		time.Sleep(time.Millisecond * 150)
		// test XMPP port...
		conn, err := net.Dial("tcp", ":5123")
		require.Nil(t, err)
		require.NotNil(t, conn)

		xmlHdr := []byte(`<?xml version="1.0" encoding="UTF-8">`)
		n, err := conn.Write(xmlHdr)
		require.Nil(t, err)
		require.Equal(t, len(xmlHdr), n)
		conn.Close()

		// test debug port...
		req, err := http.NewRequest("GET", "http://127.0.0.1:9123/debug/pprof", nil)
		require.Nil(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		Shutdown()
	}()
	cfg := config.Server{
		ID: "srv-1234",
		Transport: config.Transport{
			Type: config.Socket,
			Port: 5123,
		},
	}
	Initialize([]config.Server{cfg}, 9123)
}
