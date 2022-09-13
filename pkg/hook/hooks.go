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

package hook

import (
	"context"
	"errors"
	"math"
	"reflect"
	"sort"
	"sync"
)

// Priority defines hook execution priority.
type Priority int32

const (
	// LowestPriority defines lowest hook execution priority.
	LowestPriority = Priority(math.MinInt32)

	// LowPriority defines low hook execution priority.
	LowPriority = Priority(math.MinInt32 + 1000)

	// DefaultPriority defines default hook execution priority.
	DefaultPriority = Priority(0)

	// HighPriority defines high hook execution priority.
	HighPriority = Priority(math.MaxInt32 - 1000)

	// HighestPriority defines highest hook execution priority.
	HighestPriority = Priority(math.MaxInt32)
)

// Handler defines a generic hook handler function.
type Handler func(execCtx *ExecutionContext) error

// ErrStopped error is returned by a handler to halt hook execution.
var ErrStopped = errors.New("hook: execution stopped")

// ExecutionContext defines a hook execution info context.
type ExecutionContext struct {
	Info    interface{}
	Sender  interface{}
	Context context.Context
}

type handler struct {
	h Handler
	p Priority
}

// Hooks represents a set of module hook handlers.
type Hooks struct {
	mu       sync.RWMutex
	handlers map[string][]handler
}

// NewHooks returns a new initialized Hooks instance.
func NewHooks() *Hooks {
	return &Hooks{
		handlers: make(map[string][]handler),
	}
}

// AddHook adds a new handler to a given hook providing an execution priority value.
// hnd priority may be any number (including negative). Handlers with a higher priority are executed first.
func (h *Hooks) AddHook(hook string, hnd Handler, priority Priority) {
	h.mu.Lock()
	defer h.mu.Unlock()

	handlers := h.handlers[hook]
	handlers = append(handlers, handler{
		h: hnd, p: priority,
	})
	// sort by priority
	sort.Slice(handlers, func(i, j int) bool { return handlers[i].p > handlers[j].p })

	h.handlers[hook] = handlers
}

// RemoveHook removes a hook registered handler.
func (h *Hooks) RemoveHook(hook string, hnd Handler) {
	h.mu.Lock()
	defer h.mu.Unlock()

	handlers := h.handlers[hook]
	for i, handler := range handlers {
		if reflect.ValueOf(handler.h).Pointer() != reflect.ValueOf(hnd).Pointer() {
			continue
		}
		handlers = append(handlers[:i], handlers[i+1:]...)
		h.handlers[hook] = handlers
		return
	}
}

// Run invokes all hook handlers in order.
// If halted return value is true no more handlers are invoked.
func (h *Hooks) Run(hook string, execCtx *ExecutionContext) (halted bool, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	handlers := h.handlers[hook]
	for _, handler := range handlers {
		err := handler.h(execCtx)
		switch {
		case err == nil:
			break
		case errors.Is(err, ErrStopped):
			return true, nil
		default:
			return false, err
		}
	}
	return false, nil
}
