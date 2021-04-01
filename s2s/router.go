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

package s2s

import (
	"context"
	"errors"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/router"
)

type s2sRouter struct {
	outProvider outProvider
}

// NewRouter creates and returns an initialized S2S router.
func NewRouter(outProvider *OutProvider) router.S2SRouter {
	return &s2sRouter{
		outProvider: outProvider,
	}
}

func (r *s2sRouter) Route(ctx context.Context, stanza stravaganza.Stanza, senderDomain string) error {
	remoteJID := stanza.ToJID()
	targetDomain := remoteJID.Domain()

	stm, err := r.outProvider.GetOut(ctx, senderDomain, targetDomain)
	switch {
	case err == nil:
		break
	case errors.Is(err, errServerTimeout):
		return router.ErrRemoteServerTimeout
	default:
		return router.ErrRemoteServerNotFound
	}
	_ = stm.SendElement(stanza)
	return nil
}

func (r *s2sRouter) Start(_ context.Context) error {
	return nil
}

func (r *s2sRouter) Stop(_ context.Context) error {
	return nil
}
