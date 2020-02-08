/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

type remoteRouter struct {
}

func (r *remoteRouter) route(ctx context.Context, stanza xmpp.Stanza) error {
	return nil
}
