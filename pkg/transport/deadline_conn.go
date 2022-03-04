// Copyright 2022 The jackal Authors
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

package transport

import (
	"net"
	"time"
)

type deadlineConn struct {
	net.Conn
	connected       bool
	connTimeout     time.Duration
	rdTimeout       time.Duration
	connDeadlineHnd func()
	rdDeadlineHnd   func()
}

func newDeadlineConn(conn net.Conn, connTimeout time.Duration, readTimeout time.Duration) *deadlineConn {
	return &deadlineConn{
		Conn:        conn,
		connTimeout: connTimeout,
		rdTimeout:   readTimeout,
	}
}

func (c *deadlineConn) Read(b []byte) (n int, err error) {
	switch {
	case !c.connected && c.connDeadlineHnd != nil:
		tm := time.AfterFunc(c.connTimeout, c.connDeadlineHnd)
		n, err = c.Conn.Read(b)
		tm.Stop()
		c.connected = true

	case c.rdDeadlineHnd != nil:
		tm := time.AfterFunc(c.rdTimeout, c.rdDeadlineHnd)
		n, err = c.Conn.Read(b)
		tm.Stop()

	default:
		n, err = c.Conn.Read(b)
	}
	return
}

func (c *deadlineConn) setConnectDeadlineHandler(hnd func()) {
	c.connDeadlineHnd = hnd
}

func (c *deadlineConn) setReadDeadlineHandler(hnd func()) {
	c.rdDeadlineHnd = hnd
}

func (c *deadlineConn) underlyingConn() net.Conn {
	return c.Conn
}
