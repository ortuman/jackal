/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import "time"

const defaultRequestTimeout = time.Duration(10) * time.Second

// Config represents an cluster configuration.
type Config struct {
	Name      string
	BindPort  int
	Hosts     []string
	InTimeout time.Duration
}

type configProxy struct {
	Name      string   `yaml:"name"`
	BindPort  int      `yaml:"port"`
	Hosts     []string `yaml:"hosts"`
	InTimeout int      `yaml:"in_timeout"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.Name = p.Name
	c.BindPort = p.BindPort
	c.Hosts = p.Hosts

	c.InTimeout = time.Duration(p.InTimeout) * time.Second
	if c.InTimeout == 0 {
		c.InTimeout = defaultRequestTimeout
	}
	return nil
}
