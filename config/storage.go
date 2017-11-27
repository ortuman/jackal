/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package config

type Storage struct {
	Type  string `yaml:"type"`
	MySQL MySQL  `yaml:"mysql"`
}

type MySQL struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	PoolSize int    `yaml:"pool_size"`
}
