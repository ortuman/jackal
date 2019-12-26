/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

// vCardStorage defines storage operations for vCards
type vCardStorage interface {
	UpsertVCard(ctx context.Context, vCard xmpp.XElement, username string) error
	FetchVCard(ctx context.Context, username string) (xmpp.XElement, error)
}

// UpsertVCard inserts a new vCard element into storage,
// or updates it in case it's been previously inserted.
func UpsertVCard(ctx context.Context, vCard xmpp.XElement, username string) error {
	return instance().UpsertVCard(ctx, vCard, username)
}

// FetchVCard retrieves from storage a vCard element associated
// to a given user.
func FetchVCard(ctx context.Context, username string) (xmpp.XElement, error) {
	return instance().FetchVCard(ctx, username)
}
