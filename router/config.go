/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"crypto/tls"

	"github.com/ortuman/jackal/util"
)

const (
	defaultBindMessageBatchSize = 1000
)

// Config represents a router configuration.
type Config struct {
	BindMessageBatchSize int          `yaml:"bind_msg_batch_size"`
	Hosts                []HostConfig `yaml:"hosts"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var p configProxy
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.BindMessageBatchSize = p.BindMessageBatchSize
	c.Hosts = p.Hosts
	if c.BindMessageBatchSize == 0 {
		c.BindMessageBatchSize = defaultBindMessageBatchSize
	}
	return nil
}

type configProxy struct {
	BindMessageBatchSize int          `yaml:"bind_msg_batch_size"`
	Hosts                []HostConfig `yaml:"hosts"`
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
	cer, err := util.LoadCertificate(p.TLS.PrivKeyFile, p.TLS.CertFile, c.Name)
	if err != nil {
		return err
	}
	c.Certificate = cer
	return nil
}
