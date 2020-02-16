/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/xmpp"
)

type remoteRouter struct {
	remoteDomain string
	localDomain  string
	outProvider  s2s.OutProvider
}

func newRemoteRouter(remoteDomain, localDomain string, outProvider s2s.OutProvider) *remoteRouter {
	return &remoteRouter{
		remoteDomain: remoteDomain,
		localDomain:  localDomain,
		outProvider:  outProvider,
	}
}

func (r *remoteRouter) route(ctx context.Context, stanza xmpp.Stanza) error {
	outStm, err := r.outProvider.GetOut(ctx, r.localDomain, r.remoteDomain)
	if err != nil {
		log.Error(err)
		return router.ErrFailedRemoteConnect
	}
	outStm.SendElement(ctx, stanza)
	return nil
}
