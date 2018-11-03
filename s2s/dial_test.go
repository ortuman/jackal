/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestS2SDial(t *testing.T) {
	r, _, shutdown := setupTest(jackaDomain)
	defer shutdown()

	cfg := &Config{
		ConnectTimeout: time.Second * time.Duration(5),
		MaxStanzaSize:  8192,
		Transport: TransportConfig{
			Port:      9778,
			KeepAlive: time.Duration(600) * time.Second,
		},
	}

	// not enabled
	d := newDialer(cfg, r)

	// resolver error...
	mockedErr := errors.New("dialer mocked error")
	d.srvResolve = func(_, _, _ string) (cname string, addrs []*net.SRV, err error) {
		return "", nil, mockedErr
	}
	out, err := d.dial("jackal.im", "jabber.org")
	require.Nil(t, out)
	require.Equal(t, mockedErr, err)

	// dialer error...
	d.srvResolve = func(service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		return "", []*net.SRV{{Target: "xmpp.jabber.org", Port: 5269}}, nil
	}
	d.dialTimeout = func(_, _ string, _ time.Duration) (net.Conn, error) {
		return nil, mockedErr
	}
	out, err = d.dial("jackal.im", "jabber.org")
	require.Nil(t, out)
	require.Equal(t, mockedErr, err)

	// success
	d.dialTimeout = func(_, _ string, _ time.Duration) (net.Conn, error) {
		return newFakeSocketConn(), nil
	}
	out, err = d.dial("jackal.im", "jabber.org")
	require.NotNil(t, out)
	require.Nil(t, err)
}
