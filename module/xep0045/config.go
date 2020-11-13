/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/pkg/errors"
)

const (
	defaultServiceName = "Chatroom Server"
)

// Config represents XEP-0045 Multi-User Chat configuration
type Config struct {
	MucHost      string
	Name         string
	RoomDefaults mucmodel.RoomConfig
}

type configProxy struct {
	MucHost      string              `yaml:"host"`
	Name         string              `yaml:"name"`
	RoomDefaults mucmodel.RoomConfig `yaml:"room_defaults"`
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
	cfg.Name = p.Name
	if len(cfg.Name) == 0 {
		cfg.Name = defaultServiceName
	}
	cfg.RoomDefaults = p.RoomDefaults
	return nil
}
