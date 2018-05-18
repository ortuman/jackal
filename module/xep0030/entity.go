/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"sort"
	"sync"
)

// Feature represents a disco info feature entity.
type Feature = string

// Identity represents a disco info identity entity.
type Identity struct {
	Category string
	Type     string
	Name     string
}

// Item represents a disco info item entity.
type Item struct {
	Jid  string
	Name string
	Node string
}

// Entity represents a disco info item entity.
type Entity struct {
	mu         sync.RWMutex
	features   []Feature
	identities []Identity
	items      []Item
}

func (e *Entity) AddFeature(feature Feature) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.features = append(e.features, feature)
	sort.Slice(e.features, func(i, j int) bool { return e.features[i] < e.features[j] })
}

func (e *Entity) Features() []Feature {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.features
}

func (e *Entity) AddIdentity(identity Identity) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.identities = append(e.identities, identity)
}

func (e *Entity) Identities() []Identity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.identities
}

func (e *Entity) AddItem(item Item) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.items = append(e.items, item)
}

func (e *Entity) Items() []Item {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.items
}
