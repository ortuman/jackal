/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/router"
)

type Cluster struct {
	cfg        *Config
	memberList *memberlist.Memberlist
	router     *router.Router
}

func New(config *Config, router *router.Router) (*Cluster, error) {
	c := &Cluster{
		cfg:    config,
		router: router,
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
