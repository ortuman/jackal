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

package main

import (
	adminserver "github.com/ortuman/jackal/admin/server"
	clusterserver "github.com/ortuman/jackal/cluster/server"
)

func initAdminServer(a *serverApp, bindAddr string, port int) {
	adminSrv := adminserver.New(bindAddr, port, a.rep, a.peppers, a.sonar)
	a.registerStartStopper(adminSrv)
}

func initClusterServer(a *serverApp, bindAddr string, port int) {
	clusterSrv := clusterserver.New(bindAddr, port, a.localRouter, a.comps)
	a.registerStartStopper(clusterSrv)
	return
}
