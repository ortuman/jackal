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

package boltdb

import (
	"context"
)

type boltDBLocker struct{}

func newLockerRep() *boltDBLocker {
	return &boltDBLocker{}
}

func (l *boltDBLocker) Lock(_ context.Context, _ string) error   { return nil }
func (l *boltDBLocker) Unlock(_ context.Context, _ string) error { return nil }

// Lock satisfies repository.Locker interface.
func (r *Repository) Lock(_ context.Context, _ string) error { return nil }

// Unlock satisfies repository.Locker interface.
func (r *Repository) Unlock(_ context.Context, _ string) error { return nil }
