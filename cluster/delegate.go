/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
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
	var m model.GobDeserializer

	msgType := msg[0]
	switch msgType {
	case msgBindType:
		m = &BindMessage{}
	case msgUnbindType:
		m = &UnbindMessage{}
	case msgUpdatePresenceType:
		m = &UpdatePresenceMessage{}
	case msgRouteStanzaType:
		m = &RouteStanzaMessage{}
	default:
		log.Error(fmt.Errorf("unrecognized cluster message type: %d", msgType))
		return
	}
	dec := gob.NewDecoder(bytes.NewReader(msg[1:]))
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
