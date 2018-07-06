/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"net"
	"testing"
	"time"

	"github.com/ortuman/jackal/module"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestS2SDial(t *testing.T) {
	// s2s configuration
	cfg := Config{
		Enabled:        false,
		ConnectTimeout: time.Second * time.Duration(5),
		MaxStanzaSize:  8192,
		Transport: TransportConfig{
			Port:      9778,
			KeepAlive: time.Duration(600) * time.Second,
		},
	}

	// not enabled
	Initialize(&cfg, &module.Config{})
	out, err := GetS2SOut("jackal.im", "jabber.org")
	require.Nil(t, out)
	require.NotNil(t, err)
	Shutdown()

	resolver := func(service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		return "", []*net.SRV{{Target: "xmpp.jabber.org", Port: 5269}}, nil
	}
	mockedErr := errors.New("dialer mocked error")

	// resolver error...
	cfg.Enabled = true
	Initialize(&cfg, &module.Config{})
	defaultDialer.srvResolve = func(_, _, _ string) (cname string, addrs []*net.SRV, err error) {
		return "", nil, mockedErr
	}
	out, err = GetS2SOut("jackal.im", "jabber.org")
	require.Nil(t, out)
	require.Equal(t, mockedErr, err)
	Shutdown()

	// dialer error...
	Initialize(&cfg, &module.Config{})
	defaultDialer.srvResolve = resolver
	defaultDialer.dialTimeout = func(_, _ string, _ time.Duration) (net.Conn, error) {
		return nil, mockedErr
	}
	out, err = GetS2SOut("jackal.im", "jabber.org")
	require.Nil(t, out)
	require.Equal(t, mockedErr, err)
	Shutdown()

	// success
	Initialize(&cfg, &module.Config{})
	defaultDialer.srvResolve = resolver
	defaultDialer.dialTimeout = func(_, _ string, _ time.Duration) (net.Conn, error) {
		return newFakeSocketConn(), nil
	}
	out, err = GetS2SOut("jackal.im", "jabber.org")
	require.NotNil(t, out)
	require.Nil(t, err)
	Shutdown()
}
