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

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/transport"
)

type dialer struct {
	cfg         *Config
	router      *router.Router
	srvResolve  func(service, proto, name string) (cname string, addrs []*net.SRV, err error)
	dialTimeout func(network, address string, timeout time.Duration) (net.Conn, error)
}

func newDialer(cfg *Config, router *router.Router) *dialer {
	return &dialer{cfg: cfg, router: router, srvResolve: net.LookupSRV, dialTimeout: net.DialTimeout}
}

func (d *dialer) dial(localDomain, remoteDomain string) (*streamConfig, error) {
	_, addrs, err := d.srvResolve("xmpp-server", "tcp", remoteDomain)
	if err != nil {
		log.Warnf("srv lookup error: %v", err)
	}
	var target string

	if err != nil || len(addrs) == 1 && addrs[0].Target == "." {
		target = remoteDomain + ":5269"
	} else {
		target = strings.TrimSuffix(addrs[0].Target, ".") + ":" + strconv.Itoa(int(addrs[0].Port))
	}
	conn, err := d.dialTimeout("tcp", target, d.cfg.DialTimeout)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		ServerName:   remoteDomain,
		Certificates: d.router.Certificates(),
	}
	tr := transport.NewSocketTransport(conn, d.cfg.Transport.KeepAlive)
	return &streamConfig{
		keyGen:        &keyGen{secret: d.cfg.DialbackSecret},
		timeout:       d.cfg.Timeout,
		localDomain:   localDomain,
		remoteDomain:  remoteDomain,
		transport:     tr,
		tls:           tlsConfig,
		maxStanzaSize: d.cfg.MaxStanzaSize,
	}, nil
}
