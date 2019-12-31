/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"crypto/tls"

	utiltls "github.com/ortuman/jackal/util/tls"
	"github.com/pkg/errors"
)

// Config represents a router configuration.
type Config struct {
	Hosts []HostConfig
}

type configProxy struct {
	Hosts []HostConfig `yaml:"hosts"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	if len(p.Hosts) == 0 {
		return errors.New("empty hosts array")
	}
	c.Hosts = p.Hosts
	return nil
}

type tlsConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// HostConfig represents a host specific configuration.
type HostConfig struct {
	Name        string
	Certificate tls.Certificate
}

type hostConfigProxy struct {
	Name string    `yaml:"name"`
	TLS  tlsConfig `yaml:"tls"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *HostConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := hostConfigProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.Name = p.Name
	cer, err := utiltls.LoadCertificate(p.TLS.PrivKeyFile, p.TLS.CertFile, c.Name)
	if err != nil {
		return err
	}
	c.Certificate = cer
	return nil
}
