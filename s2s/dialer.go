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
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ortuman/jackal/log"
)

type dialer interface {
	DialContext(ctx context.Context, remoteDomain string) (net.Conn, error)
}

type srvResolveFunc func(service, proto, name string) (cname string, addrs []*net.SRV, err error)
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

type outDialer struct {
	srvResolve  srvResolveFunc
	dialContext dialFunc
}

func newDialer(timeout time.Duration) *outDialer {
	d := net.Dialer{Timeout: timeout}
	return &outDialer{
		srvResolve:  net.LookupSRV,
		dialContext: d.DialContext,
	}
}

func (d *outDialer) DialContext(ctx context.Context, remoteDomain string) (net.Conn, error) {
	_, address, err := d.srvResolve("xmpp-server", "tcp", remoteDomain)
	if err != nil {
		log.Warnf("s2s: SRV lookup error: %v", err)
	}
	var target string

	if err != nil || len(address) == 1 && address[0].Target == "." {
		target = remoteDomain + ":5269"
	} else {
		target = strings.TrimSuffix(address[0].Target, ".") + ":" + strconv.Itoa(int(address[0].Port))
	}
	conn, err := d.dialContext(ctx, "tcp", target)
	if err != nil {
		return nil, err
	}
	return conn, err
}
