/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"github.com/pkg/errors"
)

// TODO potentially add more things for configuration, e.g. can anyone create rooms?
type Config struct {
	MucHost string
}

type configProxy struct {
	MucHost string `yaml:"service"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (cfg *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	cfg.MucHost = p.MucHost
	if len(cfg.MucHost) == 0 {
		return errors.New("muc: must specify a service hostname")
	}
	return nil
}
