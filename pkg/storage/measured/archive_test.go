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

package measuredrepository

import (
	"context"
	"testing"

	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	"github.com/stretchr/testify/require"
)

func TestMeasuredArchiveRep_InsertArchiveMessage(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.InsertArchiveMessageFunc = func(ctx context.Context, message *archivemodel.Message) error {
		return nil
	}
	m := &measuredArchiveRep{rep: repMock}

	// when
	_ = m.InsertArchiveMessage(context.Background(), &archivemodel.Message{ArchiveId: "a1234"})

	// then
	require.Len(t, repMock.InsertArchiveMessageCalls(), 1)
}

func TestMeasuredArchiveRep_FetchArchiveMetadata(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchArchiveMetadataFunc = func(ctx context.Context, archiveID string) (*archivemodel.Metadata, error) {
		return nil, nil
	}
	m := &measuredArchiveRep{rep: repMock}

	// when
	_, _ = m.FetchArchiveMetadata(context.Background(), "a1234")

	// then
	require.Len(t, repMock.FetchArchiveMetadataCalls(), 1)
}

func TestMeasuredArchiveRep_FetchArchiveMessages(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchArchiveMessagesFunc = func(ctx context.Context, f *archivemodel.Filters, archiveID string) ([]*archivemodel.Message, error) {
		return nil, nil
	}
	m := &measuredArchiveRep{rep: repMock}

	// when
	_, _ = m.FetchArchiveMessages(context.Background(), &archivemodel.Filters{}, "a1234")

	// then
	require.Len(t, repMock.FetchArchiveMessagesCalls(), 1)
}

func TestMeasuredArchiveRep_DeleteArchiveOldestMessages(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteArchiveOldestMessagesFunc = func(ctx context.Context, archiveID string, maxElements int) error {
		return nil
	}
	m := &measuredArchiveRep{rep: repMock}

	// when
	err := m.DeleteArchiveOldestMessages(context.Background(), "a1234", 10)

	// then
	require.Len(t, repMock.DeleteArchiveOldestMessagesCalls(), 1)
	require.NoError(t, err)
}

func TestMeasuredArchiveRep_DeleteArchive(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteArchiveFunc = func(ctx context.Context, archiveId string) error {
		return nil
	}
	m := &measuredArchiveRep{rep: repMock}

	// when
	_ = m.DeleteArchive(context.Background(), "a1234")

	// then
	require.Len(t, repMock.DeleteArchiveCalls(), 1)
}
