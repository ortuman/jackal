/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
)

func Dial(domain string, opts stream.S2SDialerOptions) (stream.S2SOut, error) {
	_, addrs, err := net.LookupSRV("xmpp-server", "tcp", domain)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 || (len(addrs) == 1 && addrs[0].Target == ".") {
		return nil, errors.New("service not available at this domain")
	}
	target := strings.TrimSuffix(addrs[0].Target, ".")
	conn, err := net.Dial("tcp", target+":"+strconv.Itoa(int(addrs[0].Port)))
	if err != nil {
		return nil, err
	}
	tr := transport.NewSocketTransport(conn, opts.KeepAlive)
	return NewOut(domain, tr), nil
}
