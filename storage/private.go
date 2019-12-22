/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import "github.com/ortuman/jackal/xmpp"

// privateStorage defines operations for private storage
type privateStorage interface {
	FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error)
	UpsertPrivateXML(privateXML []xmpp.XElement, namespace string, username string) error
}

// FetchPrivateXML retrieves from storage a private element.
func FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	return instance().FetchPrivateXML(namespace, username)
}

// UpsertPrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	return instance().UpsertPrivateXML(privateXML, namespace, username)
}
