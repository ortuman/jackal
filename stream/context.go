/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import "sync"

// Context carries stream related variables across stream and its associated modules.
// Its methods are safe for simultaneous use by multiple goroutines.
type Context interface {
	// SetObject stores within the context an object reference.
	SetObject(object interface{}, key string)

	// Object retrieves from the context a previously stored object reference.
	Object(key string) interface{}

	// SetString stores within the context an string value.
	SetString(s string, key string)

	// String retrieves from the context a previously stored string value.
	String(key string) string

	// SetInt stores within the context an integer value.
	SetInt(integer int, key string)

	// Int retrieves from the context a previously stored integer value.
	Int(key string) int

	// SetFloat stores within the context a floating point value.
	SetFloat(float float64, key string)

	// Float retrieves from the context a previously stored floating point value.
	Float(key string) float64

	// SetBool stores within the context a boolean value.
	SetBool(boolean bool, key string)

	// Bool retrieves from the context a previously stored boolean value.
	Bool(key string) bool

	// Done returns a channel that is closed when the stream is terminated.
	Done() <-chan struct{}
}

type context struct {
	mu           sync.RWMutex
	m            map[string]interface{}
	onceHandlers map[string]struct{}
	doneCh       chan struct{}
}

// NewContext returns an initialized stream context.
func NewContext() (Context, chan<- struct{}) {
	doneCh := make(chan struct{})
	return &context{
		m:            make(map[string]interface{}),
		onceHandlers: make(map[string]struct{}),
		doneCh:       doneCh,
	}, doneCh
}

// SetObject stores within the context an object reference.
func (ctx *context) SetObject(object interface{}, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = object })
}

// Object retrieves from the context a previously stored object reference.
func (ctx *context) Object(key string) interface{} {
	var ret interface{}
	ctx.inReadLock(func() { ret = ctx.m[key] })
	return ret
}

// SetString stores within the context an string value.
func (ctx *context) SetString(s string, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = s })
}

// String retrieves from the context a previously stored string value.
func (ctx *context) String(key string) string {
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
func (ctx *context) SetInt(integer int, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = integer })
}

// Int retrieves from the context a previously stored integer value.
func (ctx *context) Int(key string) int {
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
func (ctx *context) SetFloat(float float64, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = float })
}

// Float retrieves from the context a previously stored floating point value.
func (ctx *context) Float(key string) float64 {
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
func (ctx *context) SetBool(boolean bool, key string) {
	ctx.inWriteLock(func() { ctx.m[key] = boolean })
}

// Bool retrieves from the context a previously stored boolean value.
func (ctx *context) Bool(key string) bool {
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

// Done returns a channel that is closed when the stream is terminated.
func (ctx *context) Done() <-chan struct{} {
	return ctx.doneCh
}

func (ctx *context) inWriteLock(f func()) {
	ctx.mu.Lock()
	f()
	ctx.mu.Unlock()
}

func (ctx *context) inReadLock(f func()) {
	ctx.mu.RLock()
	f()
	ctx.mu.RUnlock()
}
