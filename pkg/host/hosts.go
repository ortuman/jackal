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

package host

import (
	"crypto/tls"
	"sort"
	"sync"

	tlsutil "github.com/ortuman/jackal/pkg/util/tls"
)

const defaultDomain = "localhost"

// Hosts type represents all local domains set.
type Hosts struct {
	mu          sync.RWMutex
	defaultHost string
	hosts       map[string]tls.Certificate
}

// Configs contains a set of host configurations.
type Configs []Config

// Config contains host configuration parameters.
type Config struct {
	Domain string `fig:"domain"`
	TLS    struct {
		CertFile       string `fig:"cert_file"`
		PrivateKeyFile string `fig:"privkey_file"`
	} `fig:"tls"`
}

// NewHosts creates and initializes a Hosts instance.
func NewHosts(cfg Configs) (*Hosts, error) {
	hs := &Hosts{
		hosts: make(map[string]tls.Certificate),
	}
	if len(cfg) == 0 {
		cer, err := tlsutil.LoadCertificate("", "", defaultDomain)
		if err != nil {
			return nil, err
		}
		hs.RegisterDefaultHost(defaultDomain, cer)
		return hs, nil
	}
	for i, config := range cfg {
		cer, err := tlsutil.LoadCertificate(config.TLS.PrivateKeyFile, config.TLS.CertFile, config.Domain)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			hs.RegisterDefaultHost(config.Domain, cer)
		} else {
			hs.RegisterHost(config.Domain, cer)
		}
	}
	return hs, nil
}

// RegisterDefaultHost registers default host value along with its certificate.
func (hs *Hosts) RegisterDefaultHost(h string, cer tls.Certificate) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.defaultHost = h
	hs.hosts[h] = cer
}

// RegisterHost registers a host value along with its certificate.
func (hs *Hosts) RegisterHost(h string, cer tls.Certificate) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.hosts[h] = cer
}

// DefaultHostName returns default host name value.
func (hs *Hosts) DefaultHostName() string {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.defaultHost
}

// IsLocalHost tells whether or not d value corresponds to local host.
func (hs *Hosts) IsLocalHost(h string) bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	_, ok := hs.hosts[h]
	return ok
}

// HostNames returns the list of all registered local hosts.
func (hs *Hosts) HostNames() []string {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	var ret []string
	for n := range hs.hosts {
		ret = append(ret, n)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

// Certificates returns all registered domain certificates.
func (hs *Hosts) Certificates() []tls.Certificate {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	var certs []tls.Certificate
	for _, cer := range hs.hosts {
		certs = append(certs, cer)
	}
	return certs
}
