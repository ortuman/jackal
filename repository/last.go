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

	lastmodel "github.com/ortuman/jackal/model/last"
)

// Last defines user last activity repository operations.
type Last interface {
	// UpsertLast upserts a last activity entity into storage.
	UpsertLast(ctx context.Context, last *lastmodel.Last) error

	// FetchLast retrieves from storage last activity item associated to a user.
	FetchLast(ctx context.Context, username string) (*lastmodel.Last, error)

	// DeleteLast removes last activity item from storage.
	DeleteLast(ctx context.Context, username string) error
}
