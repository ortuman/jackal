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
	m map[string]string
}

// NewInfo returns an initialized Info instance.
func NewInfo(m map[string]string) Info {
	nm := make(map[string]string, len(m))
	for k, v := range m {
		nm[k] = v
	}
	return Info{m: nm}
}

// String returns string value associated to k key.
func (i Info) String(k string) string {
	return i.m[k]
}

// Bool returns bool value associated to k key.
func (i Info) Bool(k string) bool {
	v, _ := strconv.ParseBool(i.m[k])
	return v
}

// Int returns int value associated to k key.
func (i Info) Int(k string) int {
	v, _ := strconv.ParseInt(i.m[k], 10, strconv.IntSize)
	return int(v)
}

// Float returns float64 value associated to k key.
func (i Info) Float(k string) float64 {
	v, _ := strconv.ParseFloat(i.m[k], 64)
	return v
}

// AllKeys returns all registered info keys.
func (i Info) AllKeys() []string {
	retVal := make([]string, 0, len(i.m))
	for k := range i.m {
		retVal = append(retVal, k)
	}
	return retVal
}

// Value returns string raw value associated to k key.
func (i Info) Value(k string) (string, bool) {
	v, ok := i.m[k]
	return v, ok
}

// MutableInfo represents C2S mutable info set.
type MutableInfo struct{ Info }

// NewMutableInfo returns an initialized MutableInfo instance.
func NewMutableInfo() MutableInfo {
	return MutableInfo{
		Info: Info{m: make(map[string]string)},
	}
}

// SetString sets k to the v string value.
// Returns false in case the value is already present.
func (i MutableInfo) SetString(k string, v string) bool {
	return i.setVal(k, v)
}

// SetBool sets k to the v bool value.
// Returns false in case the value is already present.
func (i MutableInfo) SetBool(k string, v bool) bool {
	return i.setVal(k, strconv.FormatBool(v))
}

// SetInt sets k to the v int value.
// Returns false in case the value is already present.
func (i MutableInfo) SetInt(k string, v int) bool {
	return i.setVal(k, strconv.FormatInt(int64(v), 10))
}

// SetFloat sets k to the v float64 value.
// Returns false in case the value is already present.
func (i MutableInfo) SetFloat(k string, v float64) bool {
	return i.setVal(k, strconv.FormatFloat(v, 'E', -1, 64))
}

// Copy returns an info immutable copy.
func (i MutableInfo) Copy() Info {
	return NewInfo(i.m)
}

func (i MutableInfo) setVal(k, v string) bool {
	prev, ok := i.m[k]
	if ok && prev == v {
		return false
	}
	i.m[k] = v
	return true
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
