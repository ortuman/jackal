/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import "sync"

// Context carries stream related variables across stream and its associated modules.
// Its methods are safe for simultaneous use by multiple goroutines.
type Context struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

// NewContext returns an initialized stream context.
func NewContext() *Context {
	return &Context{m: make(map[string]interface{})}
}

// NewContextWithMap returns an initialized stream context by copying its values
// from a map.
func NewContextWithMap(m map[string]interface{}) *Context {
	ctxMap := make(map[string]interface{})
	for k, v := range m {
		ctxMap[k] = v
	}
	return &Context{m: ctxMap}
}

// SetString stores within the context an string value.
func (ctx *Context) SetString(s string, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = s })
}

// String retrieves from the context a previously stored string value.
func (ctx *Context) String(key string) string {
	var ret string
	ctx.inReadLock(func() {
		switch v := ctx.m[key].(type) {
		case string:
			ret = v
			return
		}
	})
	return ret
}

// SetInt stores within the context an integer value.
func (ctx *Context) SetInt(integer int, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = integer })
}

// Int retrieves from the context a previously stored integer value.
func (ctx *Context) Int(key string) int {
	var ret int
	ctx.inReadLock(func() {
		switch v := ctx.m[key].(type) {
		case int:
			ret = v
			return
		}
	})
	return ret
}

// SetFloat stores within the context a floating point value.
func (ctx *Context) SetFloat(float float64, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = float })
}

// Float retrieves from the context a previously stored floating point value.
func (ctx *Context) Float(key string) float64 {
	var ret float64
	ctx.inReadLock(func() {
		switch v := ctx.m[key].(type) {
		case float64:
			ret = v
			return
		}
	})
	return ret
}

// SetBool stores within the context a boolean value.
func (ctx *Context) SetBool(boolean bool, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = boolean })
}

// Bool retrieves from the context a previously stored boolean value.
func (ctx *Context) Bool(key string) bool {
	var ret bool
	ctx.inReadLock(func() {
		switch v := ctx.m[key].(type) {
		case bool:
			ret = v
			return
		}
	})
	return ret
}

// Map returns a stream context map copy.
func (ctx *Context) Map() map[string]interface{} {
	m := make(map[string]interface{})
	ctx.mu.RLock()
	for k, v := range ctx.m {
		m[k] = v
	}
	ctx.mu.RUnlock()
	return m
}

func (ctx *Context) inWriteLock(f func()) {
	ctx.mu.Lock()
	f()
	ctx.mu.Unlock()
}

func (ctx *Context) inReadLock(f func()) {
	ctx.mu.RLock()
	f()
	ctx.mu.RUnlock()
}
