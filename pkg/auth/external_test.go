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
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	authpb "github.com/ortuman/jackal/pkg/auth/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestExternal_Mechanism(t *testing.T) {
	// given
	e := &External{}

	// then
	require.Equal(t, "PLAIN", e.Mechanism())
}

func TestExternal_AuthenticateValidCredentials(t *testing.T) {
	// given
	clMock := &extGrpcClientMock{}
	clMock.AuthenticateFunc = func(ctx context.Context, in *authpb.AuthenticateRequest, opts ...grpc.CallOption) (*authpb.AuthenticateResponse, error) {
		if in.Username == "ortuman" && in.Password == "1234" {
			return &authpb.AuthenticateResponse{
				Authenticated: true,
			}, nil
		}
		return &authpb.AuthenticateResponse{}, nil
	}

	e := &External{cl: clMock}

	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	buf.WriteString("ortuman")
	buf.WriteByte(0)
	buf.WriteString("1234")

	auth0 := stravaganza.NewBuilder("auth").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithAttribute("mechanism", "PLAIN").
		WithText(base64.StdEncoding.EncodeToString(buf.Bytes())).
		Build()

	// when
	resp, err := e.ProcessElement(context.Background(), auth0)

	// then
	require.NotNil(t, resp)
	require.Nil(t, err)

	require.Equal(t, "success", resp.Name())

	require.True(t, e.Authenticated())
	require.Equal(t, "ortuman", e.Username())
}

func TestExternal_AuthenticateInvalidCredentials(t *testing.T) {
	// given
	clMock := &extGrpcClientMock{}
	clMock.AuthenticateFunc = func(ctx context.Context, in *authpb.AuthenticateRequest, opts ...grpc.CallOption) (*authpb.AuthenticateResponse, error) {
		if in.Username == "ortuman" && in.Password == "1234" {
			return &authpb.AuthenticateResponse{
				Authenticated: true,
			}, nil
		}
		return &authpb.AuthenticateResponse{}, nil
	}
	e := &External{cl: clMock}

	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	buf.WriteString("ortuman")
	buf.WriteByte(0)
	buf.WriteString("foo-password")

	auth0 := stravaganza.NewBuilder("auth").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithAttribute("mechanism", "PLAIN").
		WithText(base64.StdEncoding.EncodeToString(buf.Bytes())).
		Build()

	// when
	resp, err := e.ProcessElement(context.Background(), auth0)

	// then
	require.Nil(t, resp)
	require.NotNil(t, err)

	require.Equal(t, NotAuthorized, err.Reason)

	require.False(t, e.Authenticated())
	require.Equal(t, "", e.Username())
}
