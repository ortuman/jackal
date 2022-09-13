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

package repository

import (
	"context"

	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
)

// Archive defines storage operations for message archive
type Archive interface {
	// InsertArchiveMessage inserts a new message element into an archive queue.
	InsertArchiveMessage(ctx context.Context, message *archivemodel.Message) error

	// FetchArchiveMetadata returns the metadata value associated to an archive.
	FetchArchiveMetadata(ctx context.Context, archiveID string) (*archivemodel.Metadata, error)

	// FetchArchiveMessages fetches archive asscociated messages applying the passed f filters.
	FetchArchiveMessages(ctx context.Context, f *archivemodel.Filters, archiveID string) ([]*archivemodel.Message, error)

	// DeleteArchiveOldestMessages trims archive oldest messages up to a maxElements total count.
	DeleteArchiveOldestMessages(ctx context.Context, archiveID string, maxElements int) error

	// DeleteArchive clears an archive queue.
	DeleteArchive(ctx context.Context, archiveID string) error
}
