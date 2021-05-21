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

package c2smodel

import (
	"strconv"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
)

// Info represents C2S immutable info set.
type Info struct {
	M map[string]string
}

// String returns string value associated to k key.
func (i Info) String(k string) string {
	return i.M[k]
}

// Bool returns bool value associated to k key.
func (i Info) Bool(k string) bool {
	v, _ := strconv.ParseBool(i.M[k])
	return v
}

// Int returns int value associated to k key.
func (i Info) Int(k string) int {
	v, _ := strconv.ParseInt(i.M[k], 10, strconv.IntSize)
	return int(v)
}

// Float returns float64 value associated to k key.
func (i Info) Float(k string) float64 {
	v, _ := strconv.ParseFloat(i.M[k], 64)
	return v
}

// Resource represents a resource entity.
type Resource struct {
	InstanceID string
	JID        *jid.JID
	Presence   *stravaganza.Presence
	Info       Info
}

// IsAvailable returns presence available value.
func (r *Resource) IsAvailable() bool {
	if r.Presence != nil {
		return r.Presence.IsAvailable()
	}
	return false
}

// Priority returns resource presence priority.
func (r *Resource) Priority() int8 {
	if r.Presence != nil {
		return r.Presence.Priority()
	}
	return 0
}
