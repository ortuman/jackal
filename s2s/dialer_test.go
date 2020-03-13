/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDialer_Dial(t *testing.T) {
	d := newDialer()

	// resolver error...
	mockedErr := errors.New("dialer mocked error")
	d.srvResolve = func(_, _, _ string) (cname string, addrs []*net.SRV, err error) {
		return "", nil, mockedErr
	}
	out, err := d.Dial(context.Background(), "jabber.org")
	require.NotNil(t, out)
	require.Nil(t, err)

	// dialer error...
	d.srvResolve = func(service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		return "", []*net.SRV{{Target: "xmpp.jabber.org", Port: 5269}}, nil
	}
	d.dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
		return nil, mockedErr
	}
	out, err = d.Dial(context.Background(), "jabber.org")
	require.Nil(t, out)
	require.Equal(t, mockedErr, err)

	// success
	d.dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
		return newFakeSocketConn(), nil
	}
	out, err = d.Dial(context.Background(), "jabber.org")
	require.NotNil(t, out)
	require.Nil(t, err)
}
