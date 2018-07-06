/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package host

import (
	"crypto/tls"

	"github.com/ortuman/jackal/util"
)

// TLSConfig represents a host TLS configuration.
type TLSConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// Config represents a host configuration.
type Config struct {
	Name        string
	Certificate tls.Certificate
}

type configProxy struct {
	Name string    `yaml:"name"`
	TLS  TLSConfig `yaml:"tls"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.Name = p.Name
	cer, err := util.LoadCertificate(p.TLS.PrivKeyFile, p.TLS.CertFile, c.Name)
	if err != nil {
		return err
	}
	c.Certificate = cer
	return nil
}
