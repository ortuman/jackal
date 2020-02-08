/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/xmpp"
)

type s2sRouter struct {
}

func New() router.S2SRouter {
	return &s2sRouter{}
}

func (r *s2sRouter) Route(ctx context.Context, stanza xmpp.Stanza) error {
	return nil
}
