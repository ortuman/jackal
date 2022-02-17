// Copyright 2022 The jackal Authors
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

package resourcemanager

import (
	"context"

	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
)

// Manager defines generic resource manager interface.
type Manager interface {
	// PutResource registers or updates a resource into the manager.
	PutResource(ctx context.Context, res c2smodel.ResourceDesc) error

	// GetResource returns a previously registered resource.
	GetResource(ctx context.Context, username, resource string) (c2smodel.ResourceDesc, error)

	// GetResources returns all user registered resources.
	GetResources(_ context.Context, username string) ([]c2smodel.ResourceDesc, error)

	// DelResource removes a registered resource from the manager.
	DelResource(ctx context.Context, username, resource string) error

	// Start starts resource manager.
	Start(ctx context.Context) error

	// Stop stops resource manager.
	Stop(ctx context.Context) error
}
