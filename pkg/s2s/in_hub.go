// Copyright 2022 The jackal Authors
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

package s2s

import (
	"context"
	"sync"
	"time"

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"

	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/pkg/router/stream"
)

// InHub represents an S2S incoming connection hub.
type InHub struct {
	mu      sync.RWMutex
	streams map[stream.S2SInID]stream.S2SIn
	doneCh  chan chan struct{}
	logger  kitlog.Logger
}

// NewInHub creates and initializes a new InHub instance.
func NewInHub(logger kitlog.Logger) *InHub {
	return &InHub{
		streams: make(map[stream.S2SInID]stream.S2SIn),
		doneCh:  make(chan chan struct{}),
		logger:  logger,
	}
}

// Start starts InHub instance.
func (h *InHub) Start(_ context.Context) error {
	go h.reportMetrics()
	level.Info(h.logger).Log("msg", "started S2S in hub")
	return nil
}

// Stop stops InHub instance.
func (h *InHub) Stop(ctx context.Context) error {
	// stop metrics reporting
	ch := make(chan struct{})
	h.doneCh <- ch
	<-ch

	var streams []stream.S2SIn
	h.mu.RLock()
	for _, stm := range h.streams {
		streams = append(streams, stm)
	}
	h.mu.RUnlock()

	// perform stream disconnection
	errCh := make(chan error, 1)

	var wg sync.WaitGroup
	for _, s := range streams {
		wg.Add(1)
		go func(stm stream.S2SIn) {
			defer wg.Done()
			_ = stm.Disconnect(streamerror.E(streamerror.SystemShutdown))
			select {
			case <-stm.Done():
				break
			case <-ctx.Done():
				errCh <- ctx.Err()
			}
		}(s)
	}
	wg.Wait()

	var err error
	select {
	case err = <-errCh:
		break
	default:
		break
	}
	level.Info(h.logger).Log("msg", "stopped S2S in hub")
	return err
}

func (h *InHub) register(stm stream.S2SIn) {
	h.mu.Lock()
	h.streams[stm.ID()] = stm
	h.mu.Unlock()
}

func (h *InHub) unregister(stm stream.S2SIn) {
	h.mu.Lock()
	delete(h.streams, stm.ID())
	h.mu.Unlock()
}

func (h *InHub) reportMetrics() {
	tc := time.NewTicker(reportTotalConnectionsInterval)
	defer tc.Stop()

	for {
		select {
		case <-tc.C:
			h.mu.RLock()
			totalConns := len(h.streams)
			h.mu.RUnlock()
			reportTotalIncomingConnections(totalConns)

		case ch := <-h.doneCh:
			close(ch)
			return
		}
	}
}
