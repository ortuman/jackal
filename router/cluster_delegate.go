package router

import (
	"github.com/ortuman/jackal/cluster"
	"github.com/ortuman/jackal/log"
)

type clusterDelegate struct {
	r *Router
}

func (d *clusterDelegate) NodeJoined(node *cluster.Node) {
	log.Infof("join notified: %s", node.Name)
}

func (d *clusterDelegate) NodeLeft(node *cluster.Node) {
	log.Infof("leave notified: %s", node.Name)
}
