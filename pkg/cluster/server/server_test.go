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

package clusterserver

import (
	"context"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func TestServer_Listen(t *testing.T) {
	// given
	s := &Server{
		cfg: Config{
			BindAddr: "127.0.0.1",
			Port:     56000,
		},
		logger: kitlog.NewNopLogger(),
	}

	// when
	_ = s.Start(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cc, ccErr := grpc.DialContext(ctx, "127.0.0.1:56000", grpc.WithInsecure(), grpc.WithBlock())

	var ccSt connectivity.State
	if cc != nil {
		ccSt = cc.GetState()
	}

	_ = s.Stop(context.Background())

	// then
	require.Nil(t, ccErr)
	require.NotNil(t, cc)

	require.Equal(t, connectivity.Ready, ccSt)
}
