/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package host

import (
	"crypto/tls"

	utiltls "github.com/ortuman/jackal/util/tls"
)

type TLSConfig struct {
	CertFile       string `yaml:"cert_path"`
	PrivateKeyFile string `yaml:"privkey_path"`
}

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
	cer, err := utiltls.LoadCertificate(p.TLS.PrivateKeyFile, p.TLS.CertFile, c.Name)
	if err != nil {
		return err
	}
	c.Certificate = cer
	return nil
}
