/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

// Private represents an in-memory private storage.
type Private struct {
	*memoryStorage
}

// NewPrivate returns an instance of Private in-memory storage.
func NewPrivate() *Private {
	return &Private{memoryStorage: newStorage()}
}

// UpsertPrivateXML inserts a new private element into storage, or updates it in case it's been previously inserted.
func (m *Private) UpsertPrivateXML(_ context.Context, privateXML []xmpp.XElement, namespace string, username string) error {
	var priv []xmpp.Element

	// convert to concrete type
	for _, el := range privateXML {
		priv = append(priv, *xmpp.NewElementFromElement(el))
	}
	return m.saveEntities(privateStorageKey(username, namespace), &priv)
}

// FetchPrivateXML retrieves from storage a private element.
func (m *Private) FetchPrivateXML(_ context.Context, namespace string, username string) ([]xmpp.XElement, error) {
	var priv []xmpp.Element
	_, err := m.getEntities(privateStorageKey(username, namespace), &priv)
	if err != nil {
		return nil, err
	}
	var ret []xmpp.XElement
	for _, e := range priv {
		ret = append(ret, &e)
	}
	return ret, nil
}

func privateStorageKey(username, namespace string) string {
	return "privateElements:" + username + ":" + namespace
}
