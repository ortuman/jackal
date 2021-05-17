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
	"fmt"
	"sync"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/pkg/router/stream"
)

type Manager struct {
	mu     sync.RWMutex
	queues map[string]*queue
}

func NewManager() *Manager {
	return &Manager{
		queues: make(map[string]*queue),
	}
}

func (m *Manager) GetQueue(stm stream.C2S) []stravaganza.Stanza {
	q := m.getQueue(stm)
	if q == nil {
		return nil
	}
	return q.queue()
}

func (m *Manager) UnregisterQueue(stm stream.C2S) {
	m.mu.Lock()
	delete(m.queues, streamID(stm))
	m.mu.Unlock()
}

func (m *Manager) registerQueue(stm stream.C2S) error {
	q, err := newQueue(stm)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.queues[streamID(stm)] = q
	m.mu.Unlock()
	return nil
}

func (m *Manager) getQueue(stm stream.C2S) *queue {
	m.mu.RLock()
	q := m.queues[streamID(stm)]
	m.mu.RUnlock()
	return q
}

func streamID(stm stream.C2S) string {
	return fmt.Sprintf("%s/%s", stm.Username(), stm.Resource())
}
