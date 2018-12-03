/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"encoding/gob"

	"github.com/ortuman/jackal/log"
)

type memberListDelegate struct {
	cluster *Cluster
}

func (d *memberListDelegate) NodeMeta(limit int) []byte {
	return nil
}

func (d *memberListDelegate) NotifyMsg(msg []byte) {
	if len(msg) == 0 {
		return
	}
	var m Message
	dec := gob.NewDecoder(bytes.NewReader(msg))
	if err := m.FromGob(dec); err != nil {
		log.Error(err)
		return
	}
	d.cluster.handleNotifyMsg(m)
}

func (d *memberListDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (d *memberListDelegate) LocalState(join bool) []byte {
	return nil
}

func (d *memberListDelegate) MergeRemoteState(buf []byte, join bool) {
}
