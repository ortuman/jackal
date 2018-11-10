/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestS2SSocketServer(t *testing.T) {
	r, _, shutdown := setupTest(jackaDomain)
	defer shutdown()

	errCh := make(chan error)
	cfg := Config{
		ConnectTimeout: time.Second * time.Duration(5),
		MaxStanzaSize:  8192,
		Transport: TransportConfig{
			Port:      12778,
			KeepAlive: time.Duration(600) * time.Second,
		},
	}
	srv := server{cfg: &cfg, router: r, dialer: newDialer(&cfg, r)}
	go srv.start()
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

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		srv.shutdown(ctx)

		errCh <- nil
	}()
	err := <-errCh
	require.Nil(t, err)
}
