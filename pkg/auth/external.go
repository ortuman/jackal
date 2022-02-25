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
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jackal-xmpp/stravaganza"
	authpb "github.com/ortuman/jackal/pkg/auth/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/keepalive"
)

// External represents external authentication mechanism (PLAIN).
type External struct {
	address       string
	isSecure      bool
	username      string
	authenticated bool
	cc            *grpc.ClientConn
	cl            authpb.AuthenticatorClient
}

// NewExternal returns a new external authenticator.
func NewExternal(address string, isSecure bool) *External {
	return &External{
		address:  address,
		isSecure: isSecure,
	}
}

// Mechanism returns authenticator mechanism name.
func (e *External) Mechanism() string {
	return "PLAIN"
}

// Username returns authenticated username in case authentication process has been completed.
func (e *External) Username() string {
	if e.authenticated {
		return e.username
	}
	return ""
}

// Authenticated returns whether or not user has been authenticated.
func (e *External) Authenticated() bool {
	return e.authenticated
}

// UsesChannelBinding returns whether or not this authenticator requires channel binding bytes.
func (e *External) UsesChannelBinding() bool {
	return false
}

// ProcessElement process an incoming authenticator element.
func (e *External) ProcessElement(ctx context.Context, elem stravaganza.Element) (stravaganza.Element, *SASLError) {
	if len(elem.Text()) == 0 {
		return nil, newSASLError(MalformedRequest, nil)
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return nil, newSASLError(IncorrectEncoding, nil)
	}
	s := bytes.Split(b, []byte{0})
	if len(s) != 3 {
		return nil, newSASLError(IncorrectEncoding, nil)
	}
	username := string(s[1])
	password := string(s[2])

	resp, err := e.cl.Authenticate(ctx, &authpb.AuthenticateRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, newSASLError(TemporaryAuthFailure, err)
	}
	if !resp.Authenticated {
		return nil, newSASLError(NotAuthorized, nil)
	}
	e.username = username
	e.authenticated = true

	return stravaganza.NewBuilder("success").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		Build(), nil
}

// Reset resets scram internal state.
func (e *External) Reset() {
	e.username = ""
	e.authenticated = false
}

// Start dials external authenticator gRPC connection.
func (e *External) Start(ctx context.Context) error {
	var opts = []grpc.DialOption{
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 10,
			PermitWithoutStream: true,
		}),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	if !e.isSecure {
		opts = append(opts, grpc.WithInsecure())
	}
	cc, err := grpc.DialContext(ctx, e.address, opts...)
	if err != nil {
		return err
	}
	e.cc = cc
	e.cl = authpb.NewAuthenticatorClient(cc)
	return nil
}

// Stop closes underlying gRPC connection.
func (e *External) Stop(_ context.Context) error {
	return e.cc.Close()
}
