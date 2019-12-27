/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"context"

	"github.com/ortuman/jackal/cluster"
)

type clusterDelegate struct {
	r *Router
}

func (d *clusterDelegate) NotifyMessage(ctx context.Context, msg *cluster.Message) {
	d.r.handleNotifyMessage(ctx, msg)
}
func (d *clusterDelegate) NodeJoined(ctx context.Context, node *cluster.Node) {
	d.r.handleNodeJoined(ctx, node)
}
func (d *clusterDelegate) NodeUpdated(_ context.Context, _ *cluster.Node) {}

func (d *clusterDelegate) NodeLeft(ctx context.Context, node *cluster.Node) {
	d.r.handleNodeLeft(ctx, node)
}
