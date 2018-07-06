/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"crypto/tls"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/transport"
)

type dialer struct {
	cfg         *Config
	srvResolve  func(service, proto, name string) (cname string, addrs []*net.SRV, err error)
	dialTimeout func(network, address string, timeout time.Duration) (net.Conn, error)
}

func newDialer(cfg *Config) *dialer {
	return &dialer{cfg: cfg, srvResolve: net.LookupSRV, dialTimeout: net.DialTimeout}
}

func newDialerCopy(d *dialer) *dialer {
	return &dialer{cfg: d.cfg, srvResolve: d.srvResolve, dialTimeout: d.dialTimeout}
}

func (d *dialer) dial(localDomain, remoteDomain string) (*streamConfig, error) {
	_, addrs, err := d.srvResolve("xmpp-server", "tcp", remoteDomain)
	if err != nil {
		return nil, err
	}
	var target string
	if len(addrs) == 1 && addrs[0].Target == "." {
		target = remoteDomain + ":5269"
	} else {
		target = strings.TrimSuffix(addrs[0].Target, ".")
	}
	conn, err := d.dialTimeout("tcp", target+":"+strconv.Itoa(int(addrs[0].Port)), d.cfg.DialTimeout)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		ServerName:   remoteDomain,
		Certificates: host.Certificates(),
	}
	tr := transport.NewSocketTransport(conn, d.cfg.Transport.KeepAlive)
	return &streamConfig{
		keyGen:        &keyGen{d.cfg.DialbackSecret},
		localDomain:   localDomain,
		remoteDomain:  remoteDomain,
		transport:     tr,
		tls:           tlsConfig,
		maxStanzaSize: d.cfg.MaxStanzaSize,
	}, nil
}
