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

package auth

import (
	authpb "github.com/ortuman/jackal/pkg/auth/pb"
	"github.com/ortuman/jackal/pkg/storage/repository"
	"github.com/ortuman/jackal/pkg/transport"
)

//go:generate moq -out transport.mock_test.go . c2sTransport:transportMock
type c2sTransport interface {
	transport.Transport
}

//go:generate moq -out repository.mock_test.go . authUserRepository:usersRepository
type authUserRepository interface {
	repository.User
}

//go:generate moq -out ext_grpc_client.mock_test.go . extGrpcClient
type extGrpcClient interface {
	authpb.AuthenticatorClient
}
