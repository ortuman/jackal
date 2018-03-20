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

package localrouter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/router/stream"
)

// common errors
var errAlreadyRegistered = func(id stream.C2SID) error {
	return fmt.Errorf("localrouter: stream with id %s already registered", id)
}

var errStreamNotFound = func(id stream.C2SID) error {
	return fmt.Errorf("localrouter: stream with id %s not found", id)
}

// Router represents a cluster local router.
type Router struct {
	hosts hosts
	sonar *sonar.Sonar

	mu     sync.RWMutex
	stms   map[stream.C2SID]stream.C2S
	bndRes map[string]*resources
	doneCh chan chan struct{}
}

// New returns a new initialized local router.
func New(hosts *host.Hosts, sonar *sonar.Sonar) *Router {
	return &Router{
		hosts:  hosts,
		sonar:  sonar,
		stms:   make(map[stream.C2SID]stream.C2S),
		bndRes: make(map[string]*resources),
		doneCh: make(chan chan struct{}),
	}
}

// Route routes a stanza to a local router resource.
func (r *Router) Route(stanza stravaganza.Stanza, username, resource string) error {
	r.mu.RLock()
	rs := r.bndRes[username]
	r.mu.RUnlock()

	if rs == nil {
		return nil
	}
	return rs.route(stanza, resource)
}

// Disconnect performs disconnection over a local router resource.
func (r *Router) Disconnect(username, resource string, streamErr *streamerror.Error) error {
	r.mu.RLock()
	rs := r.bndRes[username]
	r.mu.RUnlock()

	if rs == nil {
		return nil
	}
	return rs.disconnect(resource, streamErr)
}

// Register registers a local router stream.
func (r *Router) Register(stm stream.C2S) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stmID := stm.ID()
	_, ok := r.stms[stmID]
	if ok {
		return errAlreadyRegistered(stmID)
	}
	r.stms[stmID] = stm
	return nil
}

// Bind binds a registered local router resource.
func (r *Router) Bind(id stream.C2SID) (stream.C2S, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stm := r.stms[id]
	if stm == nil {
		return nil, errStreamNotFound(id)
	}

	username := stm.Username()
	rs := r.bndRes[username]
	if rs == nil {
		rs = &resources{}
		r.bndRes[username] = rs
	}
	rs.bind(stm)
	delete(r.stms, id) // remove from anonymous c2s streams
	return stm, nil
}

// Unregister unregisters a local router resource.
func (r *Router) Unregister(stm stream.C2S) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.stms, stm.ID())

	username := stm.Username()
	rs := r.bndRes[username]
	if rs == nil {
		return nil
	}
	resource := stm.Resource()
	rs.unbind(resource)
	if len(rs.all()) == 0 {
		delete(r.bndRes, username)
	}
	return nil
}

// Stream returns stream identified by username and resource.
func (r *Router) Stream(username, resource string) stream.C2S {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rs := r.bndRes[username]
	if rs == nil {
		return nil
	}
	return rs.stream(resource)
}

// Start starts local router.
func (r *Router) Start(_ context.Context) error {
	go r.reportMetrics()
	return nil
}

// Stop stops local router.
func (r *Router) Stop(ctx context.Context) error {
	// stop metrics reporting
	ch := make(chan struct{})
	r.doneCh <- ch
	<-ch

	// grab all active streams
	var stms []stream.C2S

	r.mu.RLock()
	for _, stm := range r.stms {
		stms = append(stms, stm)
	}
	for _, rs := range r.bndRes {
		stms = append(stms, rs.all()...)
	}
	r.mu.RUnlock()

	// perform stream disconnection
	var wg sync.WaitGroup
	for _, s := range stms {
		wg.Add(1)
		go func(stm stream.C2S) {
			defer wg.Done()
			select {
			case <-stm.Disconnect(streamerror.E(streamerror.SystemShutdown)):
				break
			case <-ctx.Done():
				break
			}
		}(s)
	}
	wg.Wait()
	return nil
}

func (r *Router) reportMetrics() {
	tc := time.NewTicker(reportTotalConnectionsInterval)
	defer tc.Stop()

	for {
		select {
		case <-tc.C:
			r.mu.RLock()
			totalConns := len(r.stms)
			r.mu.RUnlock()
			reportTotalIncomingConnections(totalConns)

		case ch := <-r.doneCh:
			close(ch)
			return
		}
	}
}
