/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package config

import "errors"

// C2S represents a client-to-server manager configuration.
type C2S struct {
	Domains []string
}

type c2sProxyType struct {
	Domains []string `yaml:"domains"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *C2S) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := c2sProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	if len(p.Domains) == 0 {
		return errors.New("config.C2S: no domain specified")
	}
	c.Domains = p.Domains
	return nil
}
