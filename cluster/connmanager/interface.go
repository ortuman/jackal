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
	"io"
)

//go:generate moq -out localrouter.mock_test.go . LocalRouter:localRouterMock
//go:generate moq -out componentrouter.mock_test.go . ComponentRouter:componentRouterMock

//go:generate moq -out grpcconn.mock_test.go . grpcConn
type grpcConn interface {
	io.Closer
}
