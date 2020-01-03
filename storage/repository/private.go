/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

// Private defines operations for private storage
type Private interface {
	// FetchPrivateXML retrieves from storage a private element.
	FetchPrivateXML(ctx context.Context, namespace string, username string) ([]xmpp.XElement, error)

	// UpsertPrivateXML inserts a new private element into storage,
	// or updates it in case it's been previously inserted.
	UpsertPrivateXML(ctx context.Context, privateXML []xmpp.XElement, namespace string, username string) error
}
