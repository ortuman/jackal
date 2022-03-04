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

package extcomponentmanager

import (
	"context"

	clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/component"
)

//go:generate moq -out kv.mock_test.go . kvStorage:kvMock
type kvStorage interface {
	kv.KV
}

//go:generate moq -out components.mock_test.go . components
type components interface {
	RegisterComponent(ctx context.Context, comp component.Component) error
	UnregisterComponent(ctx context.Context, cHost string) error
}

//go:generate moq -out clusterconn.mock_test.go . clusterConn
type clusterConn interface {
	clusterconnmanager.Conn
}

//go:generate moq -out clusterconnmanager.mock_test.go . clusterConnManager
type clusterConnManager interface {
	GetConnection(instanceID string) (clusterconnmanager.Conn, error)
}
