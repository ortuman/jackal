/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package config

import (
	"bytes"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config represents a global configuration.
type Config struct {
	PIDFile string `yaml:"pid_path"`
	Debug   struct {
		Port int `yaml:"port"`
	} `yaml:"debug"`
	Logger  Logger   `yaml:"logger"`
	Storage Storage  `yaml:"storage"`
	C2S     C2S      `yaml:"c2s"`
	Servers []Server `yaml:"servers"`
}

// FromFile loads default global configuration from
// a specified file.
func FromFile(configFile string, cfg *Config) error {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, cfg)
}

// FromBuffer loads default global configuration from
// a specified byte buffer.
func FromBuffer(buf *bytes.Buffer, cfg *Config) error {
	return yaml.Unmarshal(buf.Bytes(), cfg)
}
