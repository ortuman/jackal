/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	PIDFile string `yaml:"pid_path"`

	Logger  Logger   `yaml:"logger"`
	Storage Storage  `yaml:"storage"`
	Servers []Server `yaml:"servers"`
}

var DefaultConfig Config

func Load(configFile string) error {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, &DefaultConfig)
}
