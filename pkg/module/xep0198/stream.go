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
	"context"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/pkg/host"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/router"
)

const streamNamespace = "urn:xmpp:sm:3"

const (
	// ModuleName represents stream module name.
	ModuleName = "stream"

	// XEPNumber represents stream XEP number.
	XEPNumber = "0198"
)

// Stream represents a stream (XEP-0198) module type.
type Stream struct {
	router router.Router
	hosts  *host.Hosts
	sn     *sonar.Sonar
}

// New returns a new initialized Stream instance.
func New(
	router router.Router,
	hosts *host.Hosts,
	sn *sonar.Sonar,
) *Stream {
	return &Stream{
		router: router,
		hosts:  hosts,
		sn:     sn,
	}
}

// Name returns stream module name.
func (m *Stream) Name() string { return ModuleName }

// StreamFeature returns stream module stream feature.
func (m *Stream) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return stravaganza.NewBuilder("sm").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		Build(), nil
}

// ServerFeatures returns stream server disco features.
func (m *Stream) ServerFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// AccountFeatures returns stream account disco features.
func (m *Stream) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// Start starts stream module.
func (m *Stream) Start(_ context.Context) error {
	log.Infow("Started stream module", "xep", XEPNumber)
	return nil
}

// Stop stops stream module.
func (m *Stream) Stop(_ context.Context) error {
	log.Infow("Stopped stream module", "xep", XEPNumber)
	return nil
}
