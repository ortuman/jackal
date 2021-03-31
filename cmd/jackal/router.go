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
	"github.com/ortuman/jackal/c2s"
	clusterrouter "github.com/ortuman/jackal/cluster/router"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/s2s"
)

func initRouters(a *serverApp) {
	// init shared resource hub
	a.resMng = c2s.NewResourceManager(a.kv)

	// init C2S router
	a.localRouter = c2s.NewLocalRouter(a.hosts, a.sonar)
	a.clusterRouter = clusterrouter.New(a.clusterConnMng)

	a.c2sRouter = c2s.NewRouter(a.localRouter, a.clusterRouter, a.resMng, a.rep, a.sonar)
	a.s2sRouter = s2s.NewRouter(a.s2sOutProvider, a.sonar)

	// init global router
	a.router = router.New(a.hosts, a.c2sRouter, a.s2sRouter)

	a.registerStartStopper(a.router)
	return
}
