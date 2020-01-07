/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	"github.com/ortuman/jackal/model"
)

// Capabilities capabilities repository operations.
type Capabilities interface {
	// UpsertCapabilities inserts capabilities associated to a node+ver pair, or updates them if previously inserted..
	UpsertCapabilities(ctx context.Context, caps *model.Capabilities) error

	// FetchCapabilities fetches capabilities associated to a give node and ver.
	FetchCapabilities(ctx context.Context, node, ver string) (*model.Capabilities, error)
}
