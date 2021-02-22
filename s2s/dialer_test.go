// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s2s

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDialer_ResolverError(t *testing.T) {
	// given
	d := newDialer(time.Minute)

	mockedErr := errors.New("dialer mocked error")
	d.srvResolve = func(_, _, _ string) (cname string, addrs []*net.SRV, err error) {
		return "", nil, mockedErr
	}

	// when
	out, err := d.DialContext(context.Background(), "jabber.org")

	// then
	require.NotNil(t, out)
	require.Nil(t, err)
}

func TestDialer_DialError(t *testing.T) {
	// given
	d := newDialer(time.Minute)

	errFoo := errors.New("foo error")
	d.srvResolve = func(service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		return "", []*net.SRV{{Target: "xmpp.jabber.org", Port: 5269}}, nil
	}
	d.dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
		return nil, errFoo
	}
	// when
	out, err := d.DialContext(context.Background(), "jabber.org")

	// then
	require.Nil(t, out)
	require.Equal(t, errFoo, err)
}

func TestDialer_Success(t *testing.T) {
	// given
	d := newDialer(time.Minute)

	conn := &netConnMock{}
	d.dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
		return conn, nil
	}
	// when
	out, err := d.DialContext(context.Background(), "jabber.org")

	// then
	require.NotNil(t, out)
	require.Nil(t, err)
}
