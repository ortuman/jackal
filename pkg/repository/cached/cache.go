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

package cachedrepository

import (
	"context"
)

// Cache defines an interface to fetch and store repository cached elements.
type Cache interface {
	// Fetch returns the bytes value associated to k key.
	// If the element is not present the returned payload will be nil.
	Fetch(ctx context.Context, k string) ([]byte, error)

	// Store stores a new element in the memory cache, overwriting it if already present.
	Store(ctx context.Context, k string, b []byte) error

	// Del removes k associated element from the memory cache.
	Del(ctx context.Context, k string) error

	// Exists returns true in case k element is present in the cache.
	Exists(ctx context.Context, k string) (bool, error)

	// Type returns a human-readable cache type name.
	Type() string
}
