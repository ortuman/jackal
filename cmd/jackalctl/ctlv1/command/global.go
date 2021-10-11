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

// copied from https://github.com/etcd-io/etcd/blob/master/etcdctl/ctlv3/command/global.go

package command

import (
	"context"
	"time"

	adminpb "github.com/ortuman/jackal/pkg/admin/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var display printer

// GlobalFlags are flags that defined globally and are inherited to all sub-commands.
type GlobalFlags struct {
	Host string

	DialTimeout    time.Duration
	CommandTimeOut time.Duration
}

func connFromCmd(cmd *cobra.Command) *grpc.ClientConn {
	dCtx, cancel := context.WithTimeout(context.Background(), dialTimeoutFromCmd(cmd))
	defer cancel()

	cc, err := grpc.DialContext(dCtx,
		hostFromCmd(cmd),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		ExitWithError(ExitError, err)
	}
	initDisplayFromCmd(cmd)
	return cc
}

func mustUsersClientFromCmd(cmd *cobra.Command) (adminpb.UsersClient, context.Context, context.CancelFunc) {
	conn := connFromCmd(cmd)
	ctx, cancel := commandCtx(cmd)
	return adminpb.NewUsersClient(conn), ctx, cancel
}

func initDisplayFromCmd(cmd *cobra.Command) {
	display = &simplePrinter{}
}

func commandCtx(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	timeOut, err := cmd.Flags().GetDuration("command-timeout")
	if err != nil {
		ExitWithError(ExitError, err)
	}
	return context.WithTimeout(context.Background(), timeOut)
}

func hostFromCmd(cmd *cobra.Command) string {
	host, err := cmd.Flags().GetString("host")
	if err != nil {
		ExitWithError(ExitError, err)
	}
	return host
}

func dialTimeoutFromCmd(cmd *cobra.Command) time.Duration {
	dialTimeout, err := cmd.Flags().GetDuration("dial-timeout")
	if err != nil {
		ExitWithError(ExitError, err)
	}
	return dialTimeout
}
