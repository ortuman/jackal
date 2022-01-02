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

package xep0092

import (
	"context"
	"os/exec"
	"strings"

	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/ortuman/jackal/pkg/version"
)

const versionNamespace = "jabber:iq:version"

var getOSInfo = func(ctx context.Context) string {
	out, _ := exec.CommandContext(ctx, "uname", "-rs").Output()
	return strings.TrimSpace(string(out))
}

const (
	// ModuleName represents version module name.
	ModuleName = "version"

	// XEPNumber represents version XEP number.
	XEPNumber = "0092"
)

// Config contains version module configuration options.
type Config struct {
	// ShowOS tells whether OS info should be revealed or not.
	ShowOS bool `fig:"show_os"`
}

// Version represents a version (XEP-0092) module type.
type Version struct {
	router router.Router
	osInfo string
	cfg    Config
}

// New returns a new initialized version instance.
func New(cfg Config, router router.Router) *Version {
	return &Version{
		router: router,
		cfg:    cfg,
	}
}

// Name returns version module name.
func (v *Version) Name() string { return ModuleName }

// StreamFeature returns version module stream feature.
func (v *Version) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns version server disco features.
func (v *Version) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{versionNamespace}, nil
}

// AccountFeatures returns version account disco features.
func (v *Version) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// MatchesNamespace tells whether namespace matches version module.
func (v *Version) MatchesNamespace(namespace string, serverTarget bool) bool {
	if !serverTarget {
		return false
	}
	return namespace == versionNamespace
}

// ProcessIQ process a version iq.
func (v *Version) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet():
		return v.getVersion(ctx, iq)
	case iq.IsSet():
		_, _ = v.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
	}
	return nil
}

// Start starts version module.
func (v *Version) Start(ctx context.Context) error {
	v.osInfo = getOSInfo(ctx)
	log.Infow("started version module", "xep", XEPNumber)
	return nil
}

// Stop stops version module.
func (v *Version) Stop(_ context.Context) error {
	log.Infow("stopped version module", "xep", XEPNumber)
	return nil
}

func (v *Version) getVersion(ctx context.Context, iq *stravaganza.IQ) error {
	q := iq.ChildNamespace("query", versionNamespace)
	if q == nil || q.ChildrenCount() > 0 {
		_, _ = v.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	// send version info
	qb := stravaganza.NewBuilder("query")
	qb.WithAttribute(stravaganza.Namespace, versionNamespace)
	qb.WithChild(
		stravaganza.NewBuilder("name").
			WithText("jackal").
			Build(),
	)
	qb.WithChild(
		stravaganza.NewBuilder("version").
			WithText(strings.TrimPrefix(version.Version.String(), "v")).
			Build(),
	)
	if v.cfg.ShowOS {
		qb.WithChild(
			stravaganza.NewBuilder("os").
				WithText(v.osInfo).
				Build(),
		)
	}
	_, _ = v.router.Route(ctx, xmpputil.MakeResultIQ(iq, qb.Build()))

	log.Infow("sent software version", "username", iq.FromJID().Node(), "resource", iq.FromJID().Resource(), "xep", XEPNumber)
	return nil
}
