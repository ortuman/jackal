/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"encoding/gob"
	"runtime"

	"github.com/ortuman/jackal/version"
)

type memberListDelegate struct {
	cluster *Cluster
}

func init() {
	gob.Register(Metadata{})
}

func (d *memberListDelegate) NodeMeta(limit int) []byte {
	var m Metadata
	m.Version = version.ApplicationVersion.String()
	m.GoVersion = runtime.Version()

	buf := bytes.NewBuffer(nil)
	gob.NewEncoder(buf).Encode(&m)
	b := make([]byte, buf.Len())
	copy(b, buf.Bytes())
	return b
}

func (d *memberListDelegate) NotifyMsg(msg []byte) {
	d.cluster.handleNotifyMsg(msg)
}

func (d *memberListDelegate) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (d *memberListDelegate) LocalState(join bool) []byte                { return nil }
func (d *memberListDelegate) MergeRemoteState(buf []byte, join bool)     {}
