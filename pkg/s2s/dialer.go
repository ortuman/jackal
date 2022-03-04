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

package s2s

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	s2sService    = "xmpp-server"
	s2sTLSService = "xmpps-server"

	outKeepAlive = time.Second * 15
)

type dialer interface {
	DialContext(ctx context.Context, remoteDomain string) (conn net.Conn, usesTLS bool, err error)
}

type srvResolveFunc func(service, proto, name string) (cname string, addrs []*net.SRV, err error)
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

type outDialer struct {
	srvResolve srvResolveFunc
	dialCtx    dialFunc
	dialTLSCtx dialFunc
}

func newDialer(timeout time.Duration, tlsCfg *tls.Config) *outDialer {
	d := net.Dialer{
		Timeout:   timeout,
		KeepAlive: outKeepAlive,
	}
	dTLS := tls.Dialer{
		NetDialer: &d,
		Config:    tlsCfg,
	}
	return &outDialer{
		srvResolve: net.LookupSRV,
		dialCtx:    d.DialContext,
		dialTLSCtx: dTLS.DialContext,
	}
}

func (d *outDialer) DialContext(ctx context.Context, remoteDomain string) (conn net.Conn, usesTLS bool, err error) {
	conn, err = d.dialSRV(ctx, remoteDomain, s2sTLSService, true)
	if err == nil {
		return conn, true, nil
	}
	conn, err = d.dialSRV(ctx, remoteDomain, s2sService, false)
	if err == nil {
		return conn, false, nil
	}
	conn, err = d.dialCtx(ctx, "tcp", net.JoinHostPort(remoteDomain, "5269"))
	return conn, false, err
}

func (d *outDialer) dialSRV(ctx context.Context, remoteDomain, service string, dialTLS bool) (net.Conn, error) {
	_, addrs, err := d.srvResolve(service, "tcp", remoteDomain)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if addr.Target == "." {
			continue
		}
		host := strings.TrimSuffix(addr.Target, ".")
		port := strconv.Itoa(int(addr.Port))

		var dialFn dialFunc
		switch dialTLS {
		case true:
			dialFn = d.dialTLSCtx
		default:
			dialFn = d.dialCtx
		}
		conn, err := dialFn(ctx, "tcp", net.JoinHostPort(host, port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, errors.New("s2s: failed to dial SRV")
}
