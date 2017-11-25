/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package config

import (
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	PIDFile string `yaml:"pid_path"`

	Logger  Logger   `yaml:"logger"`
	Storage Storage  `yaml:"storage"`
	Servers []Server `yaml:"servers"`
}

var DefaultConfig *Config

const defaultServerPort = 5222

const defaultMaxStanzaSize = 65536
const defaultConnectTimeout = 5
const defaultKeepAlive = 120

func Load(configFile string) error {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return err
	}
	c.setDefaults()
	DefaultConfig = c
	return nil
}

func (c *Config) setDefaults() {
	// logger defaults
	c.Logger.Level = strings.ToLower(c.Logger.Level)

	// storage defaults
	c.Storage.Type = strings.ToLower(c.Storage.Type)

	// server defaults
	for i := 0; i < len(c.Servers); i++ {
		// transport defaults
		c.Servers[i].Type = strings.ToLower(c.Servers[i].Type)
		c.Servers[i].Transport.Type = strings.ToLower(c.Servers[i].Transport.Type)

		if c.Servers[i].Transport.Port == 0 {
			c.Servers[i].Transport.Port = defaultServerPort
		}
		if c.Servers[i].Transport.ConnectTimeout == 0 {
			c.Servers[i].Transport.ConnectTimeout = defaultConnectTimeout
		}
		if c.Servers[i].Transport.KeepAlive == 0 {
			c.Servers[i].Transport.KeepAlive = defaultKeepAlive
		}
		if c.Servers[i].Transport.MaxStanzaSize == 0 {
			c.Servers[i].Transport.MaxStanzaSize = defaultMaxStanzaSize
		}

		// compression defaults
		c.Servers[i].Compression.Level = strings.ToLower(c.Servers[i].Compression.Level)
	}
}
