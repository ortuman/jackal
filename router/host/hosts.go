/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package host

import (
	"crypto/tls"
	"sort"

	utiltls "github.com/ortuman/jackal/util/tls"
)

const defaultDomain = "localhost"

type Hosts struct {
	defaultHostname string
	hosts           map[string]tls.Certificate
	mucHostname     string
}

func New(hostsConfig []Config) (*Hosts, error) {
	h := &Hosts{
		hosts: make(map[string]tls.Certificate),
	}
	if len(hostsConfig) > 0 {
		for i, host := range hostsConfig {
			if i == 0 {
				h.defaultHostname = host.Name
			}
			h.hosts[host.Name] = host.Certificate
		}
	} else {
		cer, err := utiltls.LoadCertificate("", "", defaultDomain)
		if err != nil {
			return nil, err
		}
		h.defaultHostname = defaultDomain
		h.hosts[defaultDomain] = cer
	}
	return h, nil
}

func (h *Hosts) DefaultHostName() string {
	return h.defaultHostname
}

func (h *Hosts) IsLocalHost(domain string) bool {
	_, ok := h.hosts[domain]
	return ok
}

func (h *Hosts) HostNames() []string {
	var ret []string
	for n := range h.hosts {
		ret = append(ret, n)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

func (h *Hosts) Certificates() []tls.Certificate {
	var certs []tls.Certificate
	for _, cer := range h.hosts {
		certs = append(certs, cer)
	}
	return certs
}

func (h *Hosts) AddMucHostname(hostname string) {
	h.mucHostname = hostname
}

func (h *Hosts) IsConferenceHost(domain string) bool {
	return domain == h.mucHostname
}
