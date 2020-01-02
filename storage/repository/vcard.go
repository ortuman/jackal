/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

// VCard defines storage operations for vCards
type VCard interface {
	UpsertVCard(ctx context.Context, vCard xmpp.XElement, username string) error
	FetchVCard(ctx context.Context, username string) (xmpp.XElement, error)
}
