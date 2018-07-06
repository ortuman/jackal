/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package host

import (
	"crypto/tls"
	"log"
	"sync"

	"github.com/ortuman/jackal/util"
)

const defaultDomain = "localhost"

var (
	instMu      sync.RWMutex
	hosts       = make(map[string]tls.Certificate)
	initialized bool
)

// Initialize initializes host manager sub system.
func Initialize(configurations []Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	if len(configurations) > 0 {
		for _, h := range configurations {
			hosts[h.Name] = h.Certificate
		}
	} else {
		cer, err := util.LoadCertificate("", "", defaultDomain)
		if err != nil {
			log.Fatalf("%v", err)
		}
		hosts[defaultDomain] = cer
	}
	initialized = true
}

// Shutdown shuts down host sub system.
func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		hosts = make(map[string]tls.Certificate)
		initialized = false
	}
}

// IsLocalHost returns true if domain is a local server domain.
func IsLocalHost(domain string) bool {
	instMu.RLock()
	defer instMu.RUnlock()
	_, ok := hosts[domain]
	return ok
}

// Certificates returns an array of all configured domain certificates.
func Certificates() []tls.Certificate {
	instMu.RLock()
	defer instMu.RUnlock()
	var certs []tls.Certificate
	for _, cer := range hosts {
		certs = append(certs, cer)
	}
	return certs
}
