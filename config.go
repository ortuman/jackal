/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package main

import (
	"bytes"
	"io/ioutil"

	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/storage"
	"gopkg.in/yaml.v2"
)

// DebugConfig represents debug server configuration.
type DebugConfig struct {
	Port int `yaml:"port"`
}

// TLSConfig represents a server TLS configuration.
type TLSConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// Config represents a global configuration.
type Config struct {
	PIDFile string         `yaml:"pid_path"`
	Debug   DebugConfig    `yaml:"debug"`
	Logger  log.Config     `yaml:"logger"`
	Storage storage.Config `yaml:"storage"`
	Hosts   []host.Config  `yaml:"hosts"`
	Modules module.Config  `yaml:"modules"`
	C2S     []c2s.Config   `yaml:"c2s"`
	S2S     *s2s.Config    `yaml:"s2s"`
}

// FromFile loads default global configuration from
// a specified file.
func (cfg *Config) FromFile(configFile string) error {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, cfg)
}

// FromBuffer loads default global configuration from
// a specified byte buffer.
func (cfg *Config) FromBuffer(buf *bytes.Buffer) error {
	return yaml.Unmarshal(buf.Bytes(), cfg)
}
