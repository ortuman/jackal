package cluster

type memberListDelegate struct {
	cluster *Cluster
}

func (d *memberListDelegate) NodeMeta(limit int) []byte {
	return nil
}

func (d *memberListDelegate) NotifyMsg(msg []byte) {
	d.cluster.handleNotifyMsg(msg)
}

func (d *memberListDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (d *memberListDelegate) LocalState(join bool) []byte {
	return nil
}

func (d *memberListDelegate) MergeRemoteState(buf []byte, join bool) {
}
