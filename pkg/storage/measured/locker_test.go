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

	"github.com/stretchr/testify/require"
)

func TestMeasuredLocker_Lock(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.LockFunc = func(ctx context.Context, lockID string) error {
		return nil
	}
	m := &measuredLocker{rep: repMock}

	// when
	_ = m.Lock(context.Background(), "l1")

	// then
	require.Len(t, repMock.LockCalls(), 1)
}

func TestMeasuredLocker_Unlock(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UnlockFunc = func(ctx context.Context, lockID string) error {
		return nil
	}
	m := &measuredLocker{rep: repMock}

	// when
	_ = m.Unlock(context.Background(), "l1")

	// then
	require.Len(t, repMock.UnlockCalls(), 1)
}
