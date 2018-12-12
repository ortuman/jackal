/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"github.com/ortuman/jackal/cluster"
)

type clusterDelegate struct {
	r *Router
}

func (d *clusterDelegate) NotifyMessage(msg *cluster.Message) {
	d.r.handleNotifyMessage(msg)
}

func (d *clusterDelegate) NodeJoined(node *cluster.Node) {
	d.r.handleNodeJoined(node)
}

func (d *clusterDelegate) NodeLeft(node *cluster.Node) {
	d.r.handleNodeLeft(node)
}
