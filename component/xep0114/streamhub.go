// Copyright 2020 The jackal Authors
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

package xep0114

import (
	"context"
	"sync"
	"time"
)

type inHub struct {
	mu      sync.RWMutex
	streams map[inComponentID]*inComponent
	doneCh  chan chan struct{}
}

func newInHub() *inHub {
	return &inHub{
		streams: make(map[inComponentID]*inComponent),
		doneCh:  make(chan chan struct{}),
	}
}

func (sh *inHub) register(stm *inComponent) {
	sh.mu.Lock()
	sh.streams[stm.id] = stm
	sh.mu.Unlock()
}

func (sh *inHub) unregister(stm *inComponent) {
	sh.mu.Lock()
	delete(sh.streams, stm.id)
	sh.mu.Unlock()
}

func (sh *inHub) start() {
	go sh.reportMetrics()
}

func (sh *inHub) stop(ctx context.Context) {
	// stop metrics reporting
	ch := make(chan struct{})
	sh.doneCh <- ch
	<-ch

	var streams []*inComponent
	sh.mu.RLock()
	for _, stm := range sh.streams {
		streams = append(streams, stm)
	}
	sh.mu.RUnlock()

	// perform stream disconnection
	var wg sync.WaitGroup
	for _, s := range streams {
		wg.Add(1)
		go func(stm *inComponent) {
			defer wg.Done()
			select {
			case <-stm.shutdown():
				break
			case <-ctx.Done():
				break
			}
		}(s)
	}
	wg.Wait()
}

func (sh *inHub) reportMetrics() {
	tc := time.NewTicker(reportTotalConnectionsInterval)
	defer tc.Stop()

	for {
		select {
		case <-tc.C:
			sh.mu.RLock()
			totalConns := len(sh.streams)
			sh.mu.RUnlock()
			reportTotalIncomingConnections(totalConns)

		case ch := <-sh.doneCh:
			close(ch)
			return
		}
	}
}
