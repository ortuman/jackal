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

package clusterconnmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/pkg/event"
	"github.com/ortuman/jackal/pkg/log"
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
	mu               sync.RWMutex
	conns            map[string]*clusterConn
	updateMembersSub sonar.SubID
	sonar            *sonar.Sonar
}

// NewManager returns a new initialized cluster connection manager.
func NewManager(sonar *sonar.Sonar) *Manager {
	return &Manager{
		sonar: sonar,
		conns: make(map[string]*clusterConn),
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

	m.updateMembersSub = m.sonar.Subscribe(event.MemberListUpdated, m.onMemberListUpdated)

	log.Infof("Started cluster connection manager")
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
	m.sonar.Unsubscribe(m.updateMembersSub)

	log.Infof("Stopped cluster connection manager... (%d total connections)", count)
	return nil
}

func (m *Manager) onMemberListUpdated(ctx context.Context, ev sonar.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inf := ev.Info().(*event.MemberListEventInfo)

	// close unregistered members connections...
	for _, instanceID := range inf.UnregisteredKeys {
		cl := m.conns[instanceID]
		if err := cl.close(); err != nil {
			log.Warnf("Failed to close cluster client conn: %s", err)
		}
		delete(m.conns, instanceID)
	}
	// dial connections to new registered members...
	for _, member := range inf.Registered {
		cl := newConn(member.Host, member.Port, member.APIVer)
		if err := cl.dialContext(ctx); err != nil {
			log.Warnf("Failed to dial cluster conn: %s", err)
			continue
		}
		log.Infof("Dialed cluster router connection at %s:%d", member.Host, member.Port)

		m.conns[member.InstanceID] = cl
	}
	return nil
}
