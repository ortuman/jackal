/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

// privateStorage defines operations for private storage
type privateStorage interface {
	FetchPrivateXML(ctx context.Context, namespace string, username string) ([]xmpp.XElement, error)
	UpsertPrivateXML(ctx context.Context, privateXML []xmpp.XElement, namespace string, username string) error
}

// FetchPrivateXML retrieves from storage a private element.
func FetchPrivateXML(ctx context.Context, namespace string, username string) ([]xmpp.XElement, error) {
	return instance().FetchPrivateXML(ctx, namespace, username)
}

// UpsertPrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdatePrivateXML(ctx context.Context, privateXML []xmpp.XElement, namespace string, username string) error {
	return instance().UpsertPrivateXML(ctx, privateXML, namespace, username)
}
