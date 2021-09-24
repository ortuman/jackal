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

	"github.com/ortuman/jackal/pkg/repository"
)

// CachedRepository is a Redis specifica cached repository type.
type CachedRepository struct {
	c   Cache
	rep repository.Repository
}

func New(cache Cache, rep repository.Repository) *CachedRepository {
	return &CachedRepository{
		c:   cache,
		rep: rep,
	}
}

// InTransaction generates a repository transaction and completes it after it's being used by f function.
// In case f returns no error tx transaction will be committed.
func (c *CachedRepository) InTransaction(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
	return c.rep.InTransaction(ctx, f)
}
