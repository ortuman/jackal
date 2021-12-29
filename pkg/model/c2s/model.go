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

// ResourceDesc represents read-only a resource description.
type ResourceDesc interface {
	// InstanceID specifies the instance identifier that registered the resource.
	InstanceID() string

	// JID returns the resource associated JID value.
	JID() *jid.JID

	// Presence returns the resource associated presence stanza.
	Presence() *stravaganza.Presence

	// Info returns resource registered info.
	Info() Info

	// IsAvailable returns presence available value.
	IsAvailable() bool

	// Priority returns resource presence priority.
	Priority() int8
}

// NewResourceDesc initializes and returns a read-only resource description.
func NewResourceDesc(instanceID string, jd *jid.JID, pr *stravaganza.Presence, inf Info) ResourceDesc {
	return &resourceDesc{
		instanceID: instanceID,
		jd:         jd,
		presence:   pr,
		info:       inf,
	}
}

type resourceDesc struct {
	instanceID string
	jd         *jid.JID
	presence   *stravaganza.Presence
	info       Info
}

func (r *resourceDesc) InstanceID() string {
	return r.instanceID
}

func (r *resourceDesc) JID() *jid.JID {
	return r.jd
}

func (r *resourceDesc) Presence() *stravaganza.Presence {
	return r.presence
}

func (r *resourceDesc) Info() Info {
	return r.info
}

func (r *resourceDesc) IsAvailable() bool {
	if r.presence != nil {
		return r.presence.IsAvailable()
	}
	return false
}

func (r *resourceDesc) Priority() int8 {
	if r.presence != nil {
		return r.presence.Priority()
	}
	return 0
}
