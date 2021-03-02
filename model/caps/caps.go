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

package capsmodel

import "github.com/jackal-xmpp/stravaganza"

// Capabilities represents presence capabilities info.
type Capabilities struct {
	Node     string
	Ver      string
	Features []string
	Form     stravaganza.Element
}

// HasFeature returns whether or not a Capabilities instance contains f feature.
func (c *Capabilities) HasFeature(f string) bool {
	for _, cf := range c.Features {
		if cf == f {
			return true
		}
	}
	return false
}
