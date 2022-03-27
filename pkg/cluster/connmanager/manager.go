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

package clusterconnmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"

	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/version"
)

var (
	// ErrConnNotFound will be returned by GetConnection in case the requested connection is not found.
	ErrConnNotFound = errors.New("clusterconnmanager: cluster connection not found")

	// ErrIncompatibleProtocol will be returned by GetConnection in case the requested connection protocol version
	// is incompatible.
	ErrIncompatibleProtocol = errors.New("clusterconnmanager: incompatible cluster API protocol")
)

// Conn defines cluster connection interface.
type Conn interface {
	LocalRouter() LocalRouter
	ComponentRouter() ComponentRouter
}

// Manager is the cluster connection manager.
type Manager struct {
	mu     sync.RWMutex
	conns  map[string]*clusterConn
	hk     *hook.Hooks
	logger kitlog.Logger
}

// NewManager returns a new initialized cluster connection manager.
func NewManager(hk *hook.Hooks, logger kitlog.Logger) *Manager {
	return &Manager{
		hk:     hk,
		conns:  make(map[string]*clusterConn),
		logger: logger,
	}
}

// GetConnection returns the connection associated to a given cluster instance.
func (m *Manager) GetConnection(instanceID string) (Conn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.conns[instanceID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrConnNotFound, instanceID)
	}
	localVer := version.ClusterAPIVersion
	remoteVer := conn.clusterAPIVer()

	if localVer.Major() != remoteVer.Major() { // we don't speak the same language
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrIncompatibleProtocol, localVer.String(), remoteVer.String())
	}
	return conn, nil
}

// Start starts cluster connection manager.
func (m *Manager) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hk.AddHook(hook.MemberListUpdated, m.onMemberListUpdated, hook.DefaultPriority)

	level.Info(m.logger).Log("msg", "started cluster connection manager")
	return nil
}

// Stop stops cluster connection manager.
func (m *Manager) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// close all cluster connections...
	count := len(m.conns)
	for instanceID, cl := range m.conns {
		if err := cl.close(); err != nil {
			return err
		}
		delete(m.conns, instanceID)
	}
	m.hk.RemoveHook(hook.MemberListUpdated, m.onMemberListUpdated)

	level.Info(m.logger).Log("msg", "stopped cluster connection manager...", "total_connections", count)
	return nil
}

func (m *Manager) onMemberListUpdated(ctx context.Context, execCtx *hook.ExecutionContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inf := execCtx.Info.(*hook.MemberListInfo)

	// close unregistered members connections...
	for _, instanceID := range inf.UnregisteredKeys {
		cl := m.conns[instanceID]
		if err := cl.close(); err != nil {
			level.Warn(m.logger).Log("msg", "failed to close cluster client conn", "err", err)
		}
		delete(m.conns, instanceID)
	}
	// dial connections to new registered members...
	for _, member := range inf.Registered {
		cl := newConn(member.Host, member.Port, member.APIVer)
		if err := cl.dialContext(ctx); err != nil {
			level.Warn(m.logger).Log("msg", "failed to dial cluster conn", "err", err)
			continue
		}
		level.Info(m.logger).Log("msg", "dialed cluster router connection", "remote_addr", fmt.Sprintf("%s:%d", member.Host, member.Port))

		m.conns[member.InstanceID] = cl
	}
	return nil
}
