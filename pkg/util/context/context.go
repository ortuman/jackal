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

package context

import "context"

type contextKey string

const (
	listenerPortKey contextKey = "ln_port"
)

// InjectListenerPort returns a ctx derived context injecting a listener port.
func InjectListenerPort(ctx context.Context, port int) context.Context {
	return context.WithValue(ctx, listenerPortKey, port)
}

// ExtractListenerPort extracts a listener port from ctx context.
func ExtractListenerPort(ctx context.Context) int {
	port, ok := ctx.Value(listenerPortKey).(int)
	if !ok {
		return 0
	}
	return port
}
