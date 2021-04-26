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

package externalmodule

import (
	"io"

	extmodulepb "github.com/ortuman/jackal/pkg/module/external/pb"
	"github.com/ortuman/jackal/pkg/router"
)

//go:generate moq -out grpc_client.mock_test.go . grpcClient
type grpcClient interface {
	extmodulepb.ModuleClient
}

//go:generate moq -out get_stanzas_client.mock_test.go . getStanzasClient
type getStanzasClient interface {
	extmodulepb.Module_GetStanzasClient
}

//go:generate moq -out closer.mock_test.go . closer
type closer interface {
	io.Closer
}

//go:generate moq -out router.mock_test.go . globalRouter:routerMock
type globalRouter interface {
	router.Router
}
