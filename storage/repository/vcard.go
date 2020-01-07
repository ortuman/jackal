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
	// UpsertVCard inserts a new vCard element into storage, or updates it in case it's been previously inserted.
	UpsertVCard(ctx context.Context, vCard xmpp.XElement, username string) error

	// FetchVCard retrieves from storage a vCard element associated to a given user.
	FetchVCard(ctx context.Context, username string) (xmpp.XElement, error)
}
