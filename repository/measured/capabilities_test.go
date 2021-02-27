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

package measuredrepository

import (
	"context"
	"testing"

	capsmodel "github.com/ortuman/jackal/model/caps"
	"github.com/stretchr/testify/require"
)

func TestMeasuredCapabilitiesRep_UpsertCapabilities(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertCapabilitiesFunc = func(ctx context.Context, caps *capsmodel.Capabilities) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{})

	// then
	require.Len(t, repMock.UpsertCapabilitiesCalls(), 1)
}

func TestMeasuredCapabilitiesRep_CapabilitiesExist(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.CapabilitiesExistFunc = func(ctx context.Context, node string, ver string) (bool, error) {
		return true, nil
	}
	m := New(repMock)

	// when
	_, _ = m.CapabilitiesExist(context.Background(), "n0", "v0")

	// then
	require.Len(t, repMock.CapabilitiesExistCalls(), 1)
}

func TestMeasuredCapabilitiesRep_FetchCapabilities(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchCapabilitiesFunc = func(ctx context.Context, node string, ver string) (*capsmodel.Capabilities, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, _ = m.FetchCapabilities(context.Background(), "n0", "v0")

	// then
	require.Len(t, repMock.FetchCapabilitiesCalls(), 1)
}
