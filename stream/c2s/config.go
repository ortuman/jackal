/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import "errors"

// Config represents a client-to-server manager configuration.
type Config struct {
	Domains []string
}

type configProxyType struct {
	Domains []string `yaml:"domains"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	if len(p.Domains) == 0 {
		return errors.New("c2s.Config: no domain specified")
	}
	c.Domains = p.Domains
	return nil
}
