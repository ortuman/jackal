/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"

	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/xmpp"
)

type remoteRouter struct {
	domain      string
	localDomain string
	outProvider s2s.OutProvider
}

func newRemoteRouter(domain, localDomain string, outProvider s2s.OutProvider) *remoteRouter {
	return &remoteRouter{
		domain:      domain,
		localDomain: localDomain,
		outProvider: outProvider,
	}
}

func (r *remoteRouter) route(ctx context.Context, stanza xmpp.Stanza) error {
	return nil
}
