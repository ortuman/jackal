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

package discomodel

// Feature represents a disco info feature entity.
type Feature = string

// Identity represents a disco info identity entity.
type Identity struct {
	Category string
	Name     string
	Type     string
	Lang     string
}

// Item represents a disco info item entity.
type Item struct {
	Jid  string
	Name string
	Node string
}
