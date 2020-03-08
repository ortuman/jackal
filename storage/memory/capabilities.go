/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	capsmodel "github.com/ortuman/jackal/model/capabilities"
)

// Capabilities represents an in-memory capabilities storage.
type Capabilities struct {
	*memoryStorage
}

// NewCapabilities returns an instance of Capabilities in-memory storage.
func NewCapabilities() *Capabilities {
	return &Capabilities{memoryStorage: newStorage()}
}

// UpsertCapabilities inserts capabilities associated to a node+ver pair, or updates them if previously inserted..
func (m *Capabilities) UpsertCapabilities(_ context.Context, caps *capsmodel.Capabilities) error {
	return m.saveEntity(capabilitiesKey(caps.Node, caps.Ver), caps)
}

// FetchCapabilities fetches capabilities associated to a give node and ver.
func (m *Capabilities) FetchCapabilities(_ context.Context, node, ver string) (*capsmodel.Capabilities, error) {
	var caps capsmodel.Capabilities

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
