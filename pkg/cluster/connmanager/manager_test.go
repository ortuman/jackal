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
	"io"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/ortuman/jackal/pkg/hook"
	clustermodel "github.com/ortuman/jackal/pkg/model/cluster"
	"github.com/ortuman/jackal/pkg/version"
	"github.com/stretchr/testify/require"
)

func TestConnections_UpdateMembers(t *testing.T) {
	// given
	lcRouterMock := &localRouterMock{}
	compRouterMock := &componentRouterMock{}
	stmMgmtMock := &streamManagementMock{}

	ccMock := &grpcConnMock{}
	ccMock.CloseFunc = func() error { return nil }

	dialFn = func(ctx context.Context, target string) (LocalRouter, ComponentRouter, StreamManagement, io.Closer, error) {
		return lcRouterMock, compRouterMock, stmMgmtMock, ccMock, nil
	}
	hk := hook.NewHooks()
	connMng := NewManager(hk, kitlog.NewNopLogger())

	// when
	_ = connMng.Start(context.Background())

	// register cluster member
	_, _ = hk.Run(hook.MemberListUpdated, &hook.ExecutionContext{
		Info: &hook.MemberListInfo{
			Registered: []clustermodel.Member{
				{InstanceID: "a1234", Host: "192.168.2.1", Port: 1234, APIVer: version.ClusterAPIVersion},
			},
		},
		Context: context.Background(),
	})

	conn1, err1 := connMng.GetConnection("a1234")

	// register cluster member
	_, _ = hk.Run(hook.MemberListUpdated, &hook.ExecutionContext{
		Info: &hook.MemberListInfo{
			UnregisteredKeys: []string{"a1234"},
		},
		Context: context.Background(),
	})

	conn2, err2 := connMng.GetConnection("a1234")

	// then
	require.Nil(t, err1)
	require.NotNil(t, conn1)

	require.Nil(t, conn2)
	require.NotNil(t, err2)

	require.True(t, errors.Is(err2, ErrConnNotFound))

	require.Len(t, ccMock.CloseCalls(), 1)
}

func TestConnections_IncompatibleClusterAPI(t *testing.T) {
	// given
	localRouterMock := &localRouterMock{}
	compRouterMock := &componentRouterMock{}
	stmMgmtMock := &streamManagementMock{}
	ccMock := &grpcConnMock{}

	dialFn = func(ctx context.Context, target string) (LocalRouter, ComponentRouter, StreamManagement, io.Closer, error) {
		return localRouterMock, compRouterMock, stmMgmtMock, ccMock, nil
	}
	hk := hook.NewHooks()
	connMng := NewManager(hk, kitlog.NewNopLogger())

	// when
	_ = connMng.Start(context.Background())

	incompVer := version.NewVersion(version.ClusterAPIVersion.Major()+1, 0, 0)
	_, _ = hk.Run(hook.MemberListUpdated, &hook.ExecutionContext{
		Info: &hook.MemberListInfo{
			Registered: []clustermodel.Member{
				{InstanceID: "a1234", Host: "192.168.2.1", Port: 1234, APIVer: incompVer},
			},
		},
		Context: context.Background(),
	})

	// then
	conn, err := connMng.GetConnection("a1234")

	require.Nil(t, conn)
	require.NotNil(t, err)

	require.True(t, errors.Is(err, ErrIncompatibleProtocol))
}
