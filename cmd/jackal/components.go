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
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/component/extcomponentmanager"
)

func initComponents(a *serverApp, cfg componentsConfig) {
	a.comps = component.NewComponents(nil)
	a.registerStartStopper(a.comps)

	a.extCompMng = extcomponentmanager.New(a.kv, a.clusterConnMng, a.comps)
	a.registerStartStopper(a.extCompMng)
}
