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

package xep0191

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
)

const blockListNamespace = "urn:xmpp:blocking"

const (
	// ModuleName represents blocklist module name.
	ModuleName = "blocklist"

	// XEPNumber represents blocklist XEP number.
	XEPNumber = "0191"
)

type BlockList struct {
	router router.Router
}

// New returns a new initialized BlockList instance.
func New(router router.Router) *BlockList {
	return &BlockList{
		router: router,
	}
}

// Name returns blocklist module name.
func (v *BlockList) Name() string { return ModuleName }

// StreamFeature returns blocklist module stream feature.
func (v *BlockList) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns blocklist server disco features.
func (v *BlockList) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{blockListNamespace}, nil
}

// AccountFeatures returns blocklist account disco features.
func (v *BlockList) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// MatchesNamespace tells whether namespace matches blocklist module.
func (v *BlockList) MatchesNamespace(namespace string, serverTarget bool) bool {
	if serverTarget {
		return false
	}
	return namespace == blockListNamespace
}

// ProcessIQ process a blocklist iq.
func (v *BlockList) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	return nil
}

// Start starts blocklist module.
func (v *BlockList) Start(ctx context.Context) error {
	log.Infow("Started blocklist module", "xep", XEPNumber)
	return nil
}

// Stop stops blocklist module.
func (v *BlockList) Stop(_ context.Context) error {
	log.Infow("Stopped blocklist module", "xep", XEPNumber)
	return nil
}
