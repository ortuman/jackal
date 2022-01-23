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

package locker

import "context"

// NewNopLocker returns a Locker that doesn't do anything.
func NewNopLocker() Locker { return &nopLocker{} }

// IsNop tells whether l is nop locker.
func IsNop(l Locker) bool {
	_, ok := l.(*nopLocker)
	return ok
}

type nopLocker struct{}

func (l *nopLocker) AcquireLock(_ context.Context, _ string) (Lock, error) {
	return &nopLock{}, nil
}

func (l *nopLocker) Start(_ context.Context) error { return nil }
func (l *nopLocker) Stop(_ context.Context) error  { return nil }

type nopLock struct{}

func (l *nopLock) Release(_ context.Context) error { return nil }
