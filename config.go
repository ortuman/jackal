/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package main

import (
	"bytes"
	"io/ioutil"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/server"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"gopkg.in/yaml.v2"
)

// Config represents a global configuration.
type Config struct {
	PIDFile string `yaml:"pid_path"`
	Debug   struct {
		Port int `yaml:"port"`
	} `yaml:"debug"`
	Logger  log.Config      `yaml:"logger"`
	Storage storage.Config  `yaml:"storage"`
	C2S     c2s.Config      `yaml:"c2s"`
	Servers []server.Config `yaml:"servers"`
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
