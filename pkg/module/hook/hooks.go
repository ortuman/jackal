// Copyright 2021 The jackal Authors
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

package modhook

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"sync"
)

// ErrExecutionHalted is returned by Run and RunFold methods to indicate the interruption of a hook execution.
var ErrExecutionHalted = errors.New("modhook: hook execution halted")

// Handler defines a generic hook handler function.
type Handler func(ctx context.Context, in Input, out *Output) error

// Input defines a hook execution input type.
type Input struct {
	Info interface{}
}

// Output defines a hook execution output type.
type Output struct {
	Info interface{}
}

type handler struct {
	h Handler
	p int32
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
func (h *Hooks) AddHook(hook string, hnd Handler, priority int32) {
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
// If a handler returns ErrExecutionHalted no more handlers are invoked.
func (h *Hooks) Run(ctx context.Context, hook string, in Input) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	handlers := h.handlers[hook]
	for _, handler := range handlers {
		if err := handler.h(ctx, in, nil); err != nil {
			return err
		}
	}
	return nil
}

// RunFold invokes all hook handlers in order propagating execution result across all invocations.
// If a handler returns ErrExecutionHalted no more handlers are invoked.
func (h *Hooks) RunFold(ctx context.Context, hook string, in Input, out *Output) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	handlers := h.handlers[hook]
	for _, handler := range handlers {
		if err := handler.h(ctx, in, out); err != nil {
			return err
		}
	}
	return nil
}
