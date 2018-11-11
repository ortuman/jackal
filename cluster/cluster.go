/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"github.com/hashicorp/memberlist"
)

type Cluster interface {
}

type cluster struct {
	cfg        *Config
	memberList *memberlist.Memberlist
}

func New(config *Config) (Cluster, error) {
	c := &cluster{
		cfg: config,
	}
	conf := memberlist.DefaultLocalConfig()
	conf.Name = config.NodeName
	conf.BindPort = config.BindPort
	ml, err := memberlist.Create(conf)
	if err != nil {
		return nil, err
	}
	c.memberList = ml
	return c, nil
}
