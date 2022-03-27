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

	lastmodel "github.com/ortuman/jackal/pkg/model/last"
	"github.com/stretchr/testify/require"
)

func TestMeasuredLastRep_UpsertLast(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertLastFunc = func(ctx context.Context, last *lastmodel.Last) error {
		return nil
	}
	m := &measuredLastRep{rep: repMock}

	// when
	_ = m.UpsertLast(context.Background(), &lastmodel.Last{
		Username: "ortuman",
		Seconds:  1000,
	})

	// then
	require.Len(t, repMock.UpsertLastCalls(), 1)
}

func TestMeasuredLastRep_FetchLast(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchLastFunc = func(ctx context.Context, username string) (*lastmodel.Last, error) {
		return &lastmodel.Last{
			Username: "ortuman",
			Seconds:  1000,
		}, nil
	}
	m := &measuredLastRep{rep: repMock}

	// when
	_, _ = m.FetchLast(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchLastCalls(), 1)
}

func TestMeasuredLastRep_DeleteLast(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteLastFunc = func(ctx context.Context, username string) error {
		return nil
	}
	m := &measuredLastRep{rep: repMock}

	// when
	_ = m.DeleteLast(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.DeleteLastCalls(), 1)
}
