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

package clusterrouter

import clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"

//go:generate moq -out clusterconn.mock_test.go . clusterConn
type clusterConn interface {
	clusterconnmanager.Conn
}

//go:generate moq -out localrouter.mock_test.go . localRouter
type localRouter interface {
	clusterconnmanager.LocalRouter
}

//go:generate moq -out componentrouter.mock_test.go . componentRouter
type componentRouter interface {
	clusterconnmanager.ComponentRouter
}

//go:generate moq -out clusterconnmanager.mock_test.go . clusterConnManager
type clusterConnManager interface {
	GetConnection(instanceID string) (clusterconnmanager.Conn, error)
}
