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

// Cache defines an interface to fetch and retrieve elements associated to a given key from a cache.
type Cache interface {
	// Get returns the bytes value associated to k.
	// If k element is not present, the returned payload will be nil.
	Get(ctx context.Context, k string) ([]byte, error)

	// Set stores a new element in the memory cache, overwriting it if it was already present.
	Set(ctx context.Context, k string, b []byte) error
}
