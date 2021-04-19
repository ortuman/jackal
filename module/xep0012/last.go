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

package xep0012

import (
	"context"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
)

const lastActivityNamespace = "jabber:iq:last"

const (
	// ModuleName represents last activity module name.
	ModuleName = "last"

	// XEPNumber represents last activity XEP number.
	XEPNumber = "0012"
)

// LastActivity represents a last activity (XEP-0012) module type.
type LastActivity struct {
	router router.Router
}

// Name returns last activity module name.
func (m *LastActivity) Name() string { return ModuleName }

// StreamFeature returns last activity stream feature.
func (m *LastActivity) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns server last activity features.
func (m *LastActivity) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{lastActivityNamespace}, nil
}

// AccountFeatures returns account last activity features.
func (m *LastActivity) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// MatchesNamespace tells whether namespace matches last activity module.
func (m *LastActivity) MatchesNamespace(namespace string, _ bool) bool {
	return namespace == lastActivityNamespace
}

// ProcessIQ process a last activity info iq.
func (m *LastActivity) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	return nil
}

// Start starts last activity module.
func (m *LastActivity) Start(_ context.Context) error {
	log.Infow("Started last module", "xep", XEPNumber)
	return nil
}

// Stop stops last activity module.
func (m *LastActivity) Stop(_ context.Context) error {
	log.Infow("Stopped last module", "xep", XEPNumber)
	return nil
}
