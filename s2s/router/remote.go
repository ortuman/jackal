/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

type remoteRouter struct {
	outStm stream.S2SOut
}

func newRemoteRouter(outStm stream.S2SOut) *remoteRouter {
	return &remoteRouter{
		outStm: outStm,
	}
}

func (r *remoteRouter) route(ctx context.Context, stanza xmpp.Stanza) error {
	r.outStm.SendElement(ctx, stanza)
	return nil
}
