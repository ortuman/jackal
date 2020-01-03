/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/model"
)

type Capabilities struct {
	*memoryStorage
}

func NewCapabilities() *Capabilities {
	return &Capabilities{memoryStorage: newStorage()}
}

func (m *Capabilities) InsertCapabilities(_ context.Context, caps *model.Capabilities) error {
	return m.saveEntity(capabilitiesKey(caps.Node, caps.Ver), caps)
}

func (m *Capabilities) FetchCapabilities(_ context.Context, node, ver string) (*model.Capabilities, error) {
	var caps model.Capabilities

	ok, err := m.getEntity(capabilitiesKey(node, ver), &caps)
	switch err {
	case nil:
		if !ok {
			return nil, nil
		}
		return &caps, nil
	default:
		return nil, err
	}
}

func capabilitiesKey(node, ver string) string {
	return "capabilities:" + node + ":" + ver
}
