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
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/ortuman/jackal/pkg/storage/repository"
	bolt "go.etcd.io/bbolt"
)

// Config contains BoltDB configuration value.
type Config struct {
	Path string `fig:"path" default:".jackal.db"`
}

// Repository represents a BoltDB repository implementation.
type Repository struct {
	repository.User
	repository.Last
	repository.Capabilities
	repository.Offline
	repository.BlockList
	repository.Private
	repository.Roster
	repository.VCard
	repository.Archive
	repository.Locker

	cfg Config

	db     *bolt.DB
	logger kitlog.Logger
}

// New creates and returns an initialized BoltDB Repository instance.
func New(cfg Config, logger kitlog.Logger) *Repository {
	return &Repository{
		cfg:    cfg,
		logger: logger,
	}
}

// InTransaction generates a BoltDB transaction and completes it after it's being used by f function.
func (r *Repository) InTransaction(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
	tx, err := r.db.Begin(true)
	if err != nil {
		return err
	}
	repTx := newRepTx(tx)
	if err := f(ctx, repTx); err != nil {
		if err := tx.Rollback(); err != nil {
			level.Warn(r.logger).Log("msg", "failed to rollback BoltDB transaction", "err", err)
		}
		return err
	}
	return tx.Commit()
}

// Start implements Start interface method.
func (r *Repository) Start(_ context.Context) error {
	db, err := bolt.Open(r.cfg.Path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return err
	}
	r.db = db

	level.Info(r.logger).Log("msg", "started BoltDB repository")
	return nil
}

// Stop closes BoltDB database.
func (r *Repository) Stop(_ context.Context) error {
	if err := r.db.Close(); err != nil {
		return err
	}
	level.Info(r.logger).Log("msg", "stopped BoltDB repository")
	return nil
}
