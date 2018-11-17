package cluster

import "github.com/hashicorp/memberlist"

type memberListEventDelegate struct {
	cluster *cluster
}

func (d *memberListEventDelegate) NotifyJoin(n *memberlist.Node) {
	d.cluster.handleNotifyJoin(n)
}

func (d *memberListEventDelegate) NotifyLeave(n *memberlist.Node) {
	d.cluster.handleNotifyLeave(n)
}

func (d *memberListEventDelegate) NotifyUpdate(n *memberlist.Node) {
	d.cluster.handleNotifyUpdate(n)
}
