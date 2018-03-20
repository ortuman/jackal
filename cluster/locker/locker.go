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

package locker

import "context"

// Locker defines distributed locking interface.
type Locker interface {
	// AcquireLock acquires lockID distributed lock.
	AcquireLock(ctx context.Context, lockID string) (Lock, error)

	// Start initializes locker.
	Start(ctx context.Context) error

	// Stop releases all locker underlying resources.
	Stop(ctx context.Context) error
}

// Lock defines distributed lock object.
type Lock interface {
	// Releases releases a previously acquired lock.
	Release(ctx context.Context) error
}
