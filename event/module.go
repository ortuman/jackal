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

package event

const (
	// ModulesStarted event is posted after initializing all configured modules.
	ModulesStarted = "modules.started"

	// ModulesStopped event is posted after finishing all configured modules.
	ModulesStopped = "modules.stopped"
)

// ModulesEventInfo contains all information associated to a modules event.
type ModulesEventInfo struct {
	ModuleNames []string
}
