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

package c2s

import (
	"sync"

	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/router/stream"
)

type resources struct {
	mu   sync.RWMutex
	stms []stream.C2S
}

func (r *resources) all() []stream.C2S {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stms
}

func (r *resources) len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.stms)
}

func (r *resources) bind(stm stream.C2S) {
	r.mu.Lock()
	defer r.mu.Unlock()

	res := stm.Resource()
	for _, s := range r.stms {
		if s.Resource() == res {
			return
		}
	}
	r.stms = append(r.stms, stm)
}

func (r *resources) unbind(res string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, s := range r.stms {
		if s.Resource() != res {
			continue
		}
		r.stms = append(r.stms[:i], r.stms[i+1:]...)
		return
	}
}

func (r *resources) stream(res string) stream.C2S {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.stms {
		if s.Resource() == res {
			return s
		}
	}
	return nil
}

func (r *resources) route(stanza stravaganza.Stanza, resource string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.stms {
		if s.Resource() != resource {
			continue
		}
		s.SendElement(stanza)
	}
	return nil
}

func (r *resources) disconnect(resource string, streamErr *streamerror.Error) error {
	var stm stream.C2S

	// grab resource stream
	r.mu.RLock()
	for _, s := range r.stms {
		if s.Resource() != resource {
			continue
		}
		stm = s
		break
	}
	r.mu.RUnlock()

	if stm == nil {
		return nil
	}
	return <-stm.Disconnect(streamErr)
}
