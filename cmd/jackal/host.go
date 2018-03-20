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

package main

import (
	"github.com/ortuman/jackal/host"
	tlsutil "github.com/ortuman/jackal/util/tls"
)

func initHosts(a *serverApp, configs []hostConfig) error {
	h := host.New()
	if len(configs) == 0 {
		cer, err := tlsutil.LoadCertificate("", "", defaultDomain)
		if err != nil {
			return err
		}
		h.RegisterDefaultHost(defaultDomain, cer)
		a.hosts = h
		return nil
	}
	for i, config := range configs {
		cer, err := tlsutil.LoadCertificate(config.TLS.PrivateKeyFile, config.TLS.CertFile, config.Domain)
		if err != nil {
			return err
		}
		if i == 0 {
			h.RegisterDefaultHost(config.Domain, cer)
		} else {
			h.RegisterHost(config.Domain, cer)
		}
	}
	a.hosts = h
	return nil
}
