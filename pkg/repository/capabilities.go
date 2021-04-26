// Copyright 2021 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"context"

	capsmodel "github.com/ortuman/jackal/pkg/model/caps"
)

// Capabilities defines user capabilities repository operations.
type Capabilities interface {
	// UpsertCapabilities upserts capabilities associated to a node+ver pair.
	UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error

	// CapabilitiesExist tells whether node+ver capabilities have been already registered.
	CapabilitiesExist(ctx context.Context, node, ver string) (bool, error)

	// FetchCapabilities fetches capabilities associated to a given node+ver pair.
	FetchCapabilities(ctx context.Context, node, ver string) (*capsmodel.Capabilities, error)
}
