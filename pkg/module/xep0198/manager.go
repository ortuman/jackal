// Copyright 2021 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0198

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"sync"

	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const nonceLength = 24

var errInvalidSMID = errors.New("xep0198: invalid stream identifier format")

type manager struct {
	mu     sync.RWMutex
	queues map[string]*stmQ
}

func newManager() *manager {
	return &manager{
		queues: make(map[string]*stmQ),
	}
}

func (m *manager) unregister(stm stream.C2S) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sID := stmID(stm.Username(), stm.Resource())
	sq := m.queues[sID]
	if sq == nil {
		return
	}
	sq.cancelTimers()
	delete(m.queues, sID)
}

func (m *manager) register(stm stream.C2S) (smID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sID := stmID(stm.Username(), stm.Resource())
	_, ok := m.queues[sID]
	if ok {
		return "", fmt.Errorf("xep0198: stream already registered: %s", sID)
	}
	// generate nonce
	nonce := make([]byte, nonceLength)
	for i := range nonce {
		nonce[i] = byte(rand.Intn(255) + 1)
	}
	m.queues[sID] = newSQ(stm, nonce)

	return encodeSMID(stm.JID(), nonce), nil
}

func (m *manager) getQueue(stm stream.C2S) *stmQ {
	m.mu.RLock()
	q := m.queues[stmID(stm.Username(), stm.Resource())]
	m.mu.RUnlock()
	return q
}

func encodeSMID(jd *jid.JID, nonce []byte) string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(jd.String())
	buf.WriteByte(0)
	buf.Write(nonce)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func decodeSMID(smID string) (jd *jid.JID, nonce []byte, err error) {
	b, err := base64.StdEncoding.DecodeString(smID)
	if err != nil {
		return nil, nil, err
	}
	ss := bytes.Split(b, []byte{0})
	if len(ss) != 2 {
		return nil, nil, errInvalidSMID
	}
	jd, err = jid.NewWithString(string(ss[0]), false)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errInvalidSMID, err)
	}
	return jd, ss[1], nil
}

func stmID(username, resource string) string {
	return fmt.Sprintf("%s/%s", username, resource)
}
