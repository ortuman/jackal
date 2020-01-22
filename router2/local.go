/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router2

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

type localRouter struct {
}

func (r *localRouter) Route(ctx context.Context, stanza xmpp.Stanza) error {
	// TODO(ortuman): implement me!
	return nil
}
