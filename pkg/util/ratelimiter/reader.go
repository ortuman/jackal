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

package ratelimiter

import (
	"errors"
	"io"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// ErrReadLimitExcedeed will be returned by Read method when current rate limit is exceeded.
var ErrReadLimitExcedeed = errors.New("ratelimiter: read limit exceeded")

// Reader implements io.Reader interface.
type Reader struct {
	r    io.Reader
	rLim atomic.Value
}

// NewReader returns a rate limited io.Read implementation.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Read implements Reader interface method.
func (lr *Reader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	if err != nil {
		return 0, err
	}
	if v := lr.rLim.Load(); v != nil {
		rLim := v.(*rate.Limiter)
		if !rLim.AllowN(time.Now(), n) {
			return 0, ErrReadLimitExcedeed
		}
	}
	return n, nil
}

// SetReadRateLimiter sets current ReadWriter read rate limit.
func (lr *Reader) SetReadRateLimiter(rLim *rate.Limiter) {
	lr.rLim.Store(rLim)
}

// ReadRateLimiter returns previously set rate limiter.
func (lr *Reader) ReadRateLimiter() *rate.Limiter {
	if v := lr.rLim.Load(); v != nil {
		return v.(*rate.Limiter)
	}
	return nil
}
