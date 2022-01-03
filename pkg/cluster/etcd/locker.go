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

package etcd

import (
	"context"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/ortuman/jackal/pkg/cluster/locker"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type etcdLock struct {
	mu *concurrency.Mutex
}

func (m *etcdLock) Release(ctx context.Context) error { return m.mu.Unlock(ctx) }

// Locker defines etcd locker.Locker implementation.
type Locker struct {
	cfg    Config
	logger kitlog.Logger
	cli    *etcdv3.Client
	ss     *concurrency.Session
}

// NewLocker returns a new initialized etcd locker.
func NewLocker(cfg Config, logger kitlog.Logger) *Locker {
	return &Locker{cfg: cfg, logger: logger}
}

// AcquireLock acquires and returns an etcd locker.
func (l *Locker) AcquireLock(ctx context.Context, lockID string) (locker.Lock, error) {
	mu := concurrency.NewMutex(l.ss, lockID)
	if err := mu.Lock(ctx); err != nil {
		return nil, err
	}
	return &etcdLock{mu: mu}, nil
}

// Start starts etcd locker.
func (l *Locker) Start(_ context.Context) error {
	// perform dialing
	cli, err := dial(l.cfg)
	if err != nil {
		return err
	}
	l.cli = cli

	ss, err := concurrency.NewSession(l.cli)
	if err != nil {
		return err
	}
	l.ss = ss
	level.Info(l.logger).Log("msg", "started etcd locker")
	return nil
}

// Stop stops etcd locker.
func (l *Locker) Stop(_ context.Context) error {
	if err := l.ss.Close(); err != nil {
		return err
	}
	if err := l.cli.Close(); err != nil {
		return err
	}
	level.Info(l.logger).Log("msg", "stopped etcd locker")
	return nil
}
