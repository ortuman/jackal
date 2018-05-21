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

// AddFeature adds a new disco entity feature.
func (e *Entity) AddFeature(feature Feature) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.features = append(e.features, feature)
	sort.Slice(e.features, func(i, j int) bool { return e.features[i] < e.features[j] })
}

// RemoveFeature removes a disco feature from entity.
func (e *Entity) RemoveFeature(feature Feature) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, f := range e.features {
		if f == feature {
			e.features = append(e.features[:i], e.features[i+1:]...)
			return
		}
	}
}

// Features returns disco entity features.
func (e *Entity) Features() []Feature {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.features
}

// AddIdentity adds a new disco entity identity.
func (e *Entity) AddIdentity(identity Identity) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.identities = append(e.identities, identity)
}

// RemoveIdentity removes a disco identity from entity.
func (e *Entity) RemoveIdentity(identity Identity) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, idn := range e.identities {
		if idn.Type == identity.Type && idn.Category == identity.Category && idn.Name == identity.Name {
			e.identities = append(e.identities[:i], e.identities[i+1:]...)
			return
		}
	}
}

// Identities returns disco entity identities.
func (e *Entity) Identities() []Identity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.identities
}

// AddItem adds a new disco entity item.
func (e *Entity) AddItem(item Item) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.items = append(e.items, item)
}

// RemoveItem removes a disco item from entity.
func (e *Entity) RemoveItem(item Item) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, itm := range e.items {
		if itm.Jid == item.Jid && itm.Node == item.Node && itm.Name == item.Name {
			e.items = append(e.items[:i], e.items[i+1:]...)
			return
		}
	}
}

// Items returns disco entity items.
func (e *Entity) Items() []Item {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.items
}
