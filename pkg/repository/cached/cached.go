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

	"github.com/ortuman/jackal/pkg/log"

	"github.com/ortuman/jackal/pkg/repository"
)

// CachedRepository is a Redis specifica cached repository type.
type CachedRepository struct {
	repository.User
	repository.VCard
	repository.Last
	repository.Capabilities
	repository.Offline
	repository.BlockList
	repository.Private
	repository.Roster

	c   Cache
	rep repository.Repository
}

// New returns an initialized CachedRepository instance.
func New(cache Cache, rep repository.Repository) *CachedRepository {
	return &CachedRepository{
		User:         &cachedUserRepository{c: cache, baseRep: rep},
		VCard:        rep,
		Last:         rep,
		Capabilities: rep,
		Offline:      rep,
		BlockList:    rep,
		Private:      rep,
		Roster:       rep,

		c:   cache,
		rep: rep,
	}
}

// InTransaction generates a repository transaction and completes it after it's being used by f function.
// In case f returns no error tx transaction will be committed.
func (c *CachedRepository) InTransaction(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
	err := c.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		return f(ctx, newCachedTx(tx))
	})
	return err
}

// Start initializes repository.
func (c *CachedRepository) Start(ctx context.Context) error {
	log.Infow("Started cached repository", "cache_type", c.c.Type())
	return c.rep.Start(ctx)
}

// Stop releases all underlying repository resources.
func (c *CachedRepository) Stop(ctx context.Context) error {
	log.Infow("Stopped cached repository", "cache_type", c.c.Type())
	return c.rep.Stop(ctx)
}
