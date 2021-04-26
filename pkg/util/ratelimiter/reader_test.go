// Copyright 2020 The jackal Authors
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

package ratelimiter

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

type mockReader struct {
	rFn func(p []byte) (n int, err error)
}

func (mr *mockReader) Read(p []byte) (n int, err error) { return mr.rFn(p) }

func TestReader_ReadRateLimit(t *testing.T) {
	// given
	mockR := &mockReader{}
	mockR.rFn = func(p []byte) (n int, err error) {
		p[0] = 0x23
		return 1, nil
	}
	r := NewReader(mockR)

	p := make([]byte, 1)

	// when
	var err1, err2 error

	// no rate limit
	for i := 0; i < 50_000; i++ {
		_, err1 = r.Read(p)
		if err1 != nil {
			break
		}
	}
	// 25k/s rate limit
	r.SetReadRateLimiter(rate.NewLimiter(25_000, 0))
	for i := 0; i < 50_000; i++ {
		_, err2 = r.Read(p)
		if err2 != nil {
			break
		}
	}

	// then
	require.Nil(t, err1)
	require.NotNil(t, err2)

	require.Equal(t, ErrReadLimitExcedeed, err2)
}
