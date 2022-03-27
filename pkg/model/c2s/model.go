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

package c2smodel

import (
	"strconv"
	"sync"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
)

// Info represents C2S immutable info set.
type Info interface {
	// String returns string value associated to k key.
	String(k string) string

	// Bool returns bool value associated to k key.
	Bool(k string) bool

	// Int returns int value associated to k key.
	Int(k string) int

	// Float returns float64 value associated to k key.
	Float(k string) float64

	// Map returns map info set copy.
	Map() map[string]string
}

type infoMap struct {
	mu sync.RWMutex
	m  map[string]string
}

func (im *infoMap) String(k string) string {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.m[k]
}

func (im *infoMap) Bool(k string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()
	v, _ := strconv.ParseBool(im.m[k])
	return v
}

func (im *infoMap) Int(k string) int {
	im.mu.RLock()
	defer im.mu.RUnlock()
	v, _ := strconv.ParseInt(im.m[k], 10, strconv.IntSize)
	return int(v)
}

func (im *infoMap) Float(k string) float64 {
	im.mu.RLock()
	defer im.mu.RUnlock()
	v, _ := strconv.ParseFloat(im.m[k], 64)
	return v
}

func (im *infoMap) Map() map[string]string {
	im.mu.RLock()
	defer im.mu.RUnlock()
	retVal := make(map[string]string, len(im.m))
	for k, v := range im.m {
		retVal[k] = v
	}
	return retVal
}

// InfoMap represents a mutable c2s info set.
type InfoMap struct {
	infoMap
}

// SetString sets a string info value.
func (im *InfoMap) SetString(k, v string) {
	im.infoMap.mu.Lock()
	im.m[k] = v
	im.infoMap.mu.Unlock()
}

// SetBool sets a boolean info value.
func (im *InfoMap) SetBool(k string, v bool) {
	im.infoMap.mu.Lock()
	im.m[k] = strconv.FormatBool(v)
	im.infoMap.mu.Unlock()
}

// SetInt sets an integer info value.
func (im *InfoMap) SetInt(k string, v int) {
	im.infoMap.mu.Lock()
	im.m[k] = strconv.Itoa(v)
	im.infoMap.mu.Unlock()
}

// SetFloat sets a float info value.
func (im *InfoMap) SetFloat(k string, v float64) {
	im.infoMap.mu.Lock()
	im.m[k] = strconv.FormatFloat(v, 'E', -1, 64)
	im.infoMap.mu.Unlock()
}

// ReadOnly returns a read-only info map reference.
func (im *InfoMap) ReadOnly() Info {
	return &im.infoMap
}

// NewInfoMap returns an empty mutable info set.
func NewInfoMap() *InfoMap {
	return &InfoMap{
		infoMap: infoMap{
			m: make(map[string]string),
		},
	}
}

// NewInfoMapFromMap returns a mutable info set of derived from m.
func NewInfoMapFromMap(m map[string]string) *InfoMap {
	retVal := &InfoMap{
		infoMap: infoMap{
			m: make(map[string]string, len(m)),
		},
	}
	for k, v := range m {
		retVal.m[k] = v
	}
	return retVal
}

// NewInfoMapFromInfo returns a mutable info set derived from a read-only info map.
func NewInfoMapFromInfo(inf Info) *InfoMap {
	var m map[string]string
	switch im := inf.(type) {
	case *InfoMap:
		m = im.m
	case *infoMap:
		m = im.m
	}
	return NewInfoMapFromMap(m)
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
