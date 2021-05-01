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

package coremodel

import (
	"fmt"
	"strconv"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/version"
)

// ClusterMember represents a cluster instance address and port.
type ClusterMember struct {
	InstanceID string
	Host       string
	Port       int
	APIVer     *version.SemanticVersion
}

// String returns Member string representation.
func (m *ClusterMember) String() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

// User represents a user entity.
type User struct {
	Username string
	Scram    struct {
		SHA1           string
		SHA256         string
		SHA512         string
		SHA3512        string
		Salt           string
		IterationCount int
		PepperID       string
	}
}

// Resource represents a resource entity.
type Resource struct {
	InstanceID string
	JID        *jid.JID
	Presence   *stravaganza.Presence
	Info       ResourceInfo
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

// ResourceInfo represents a resource user info.
type ResourceInfo struct {
	M map[string]string
}

// String returns string value associated to k key.
func (ui *ResourceInfo) String(k string) string {
	return ui.M[k]
}

// Bool returns boolean value associated to k key.
func (ui *ResourceInfo) Bool(k string) bool {
	ok, _ := strconv.ParseBool(ui.M[k])
	return ok
}

// Int returns integer value associated to k key.
func (ui *ResourceInfo) Int(k string) int64 {
	i, _ := strconv.ParseInt(ui.M[k], 10, 64)
	return i
}

// Float returns integer value associated to k key.
func (ui *ResourceInfo) Float(k string) float64 {
	f, _ := strconv.ParseFloat(ui.M[k], 64)
	return f
}
