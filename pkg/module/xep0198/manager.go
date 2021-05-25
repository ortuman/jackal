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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const nonceLength = 16

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

	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}
	m.queues[sID] = newSQ(stm, nonce)

	return encodeSMID(stm.Username(), stm.Resource(), nonce), nil
}

func (m *manager) getQueue(stm stream.C2S) *stmQ {
	m.mu.RLock()
	q := m.queues[stmID(stm.Username(), stm.Resource())]
	m.mu.RUnlock()
	return q
}

func stmID(username, resource string) string {
	return fmt.Sprintf("%s/%s", username, resource)
}

func encodeSMID(username, resource string, nonce []byte) string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(fmt.Sprintf("%s/%s", username, resource))
	buf.WriteByte(0)
	buf.Write(nonce)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func decodeSMID(smID string) (jd *jid.JID, nonce []byte, err error) {
	return nil, nil, nil
}
