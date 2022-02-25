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

package xep0202

import (
	"context"
	"time"

	"github.com/go-kit/log/level"

	kitlog "github.com/go-kit/log"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const timeNamespace = "urn:xmpp:time"

const (
	// ModuleName represents time module name.
	ModuleName = "time"

	// XEPNumber represents time XEP number.
	XEPNumber = "0202"
)

// Time represents a last activity (XEP-0202) module type.
type Time struct {
	router router.Router
	tmFn   func() time.Time
	logger kitlog.Logger
}

// New returns a new initialized Time instance.
func New(
	router router.Router,
	logger kitlog.Logger,
) *Time {
	return &Time{
		router: router,
		tmFn:   time.Now,
		logger: kitlog.With(logger, "module", ModuleName, "xep", XEPNumber),
	}
}

// Name returns time module name.
func (m *Time) Name() string { return ModuleName }

// StreamFeature returns time module stream feature.
func (m *Time) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns time server disco features.
func (m *Time) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{timeNamespace}, nil
}

// AccountFeatures returns time account disco features.
func (m *Time) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// Start starts time module.
func (m *Time) Start(_ context.Context) error {
	level.Info(m.logger).Log("msg", "started time module")
	return nil
}

// Stop stops time module.
func (m *Time) Stop(_ context.Context) error {
	level.Info(m.logger).Log("msg", "stopped time module")
	return nil
}

// MatchesNamespace tells whether namespace matches time module.
func (m *Time) MatchesNamespace(namespace string, serverTarget bool) bool {
	if !serverTarget {
		return false
	}
	return namespace == timeNamespace
}

// ProcessIQ process a time iq.
func (m *Time) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet() && iq.ChildNamespace("time", timeNamespace) != nil:
		m.reportServerTime(ctx, iq)
		return nil
	default:
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
}

func (m *Time) reportServerTime(ctx context.Context, iq *stravaganza.IQ) {
	tm := m.tmFn()

	resIQ := xmpputil.MakeResultIQ(iq, stravaganza.NewBuilder("time").
		WithAttribute(stravaganza.Namespace, timeNamespace).
		WithChild(stravaganza.NewBuilder("tzo").WithText(tm.Format("-07:00")).Build()).
		WithChild(stravaganza.NewBuilder("utc").WithText(tm.UTC().Format("2006-01-02T15:04:05Z")).Build()).
		Build(),
	)
	_, _ = m.router.Route(ctx, resIQ)
}
