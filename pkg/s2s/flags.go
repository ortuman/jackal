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

package s2s

import "sync"

const (
	fSecured               uint8 = 1 << 0
	fAuthenticated               = 1 << 2
	fDialbackKeyAuthorized       = 1 << 3
)

type flags struct {
	mtx sync.RWMutex
	fs  uint8
}

func (f *flags) isSecured() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.fs&fSecured > 0
}

func (f *flags) setSecured() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.fs = f.fs | fSecured
}

func (f *flags) isAuthenticated() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.fs&fAuthenticated > 0
}

func (f *flags) setAuthenticated() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.fs = f.fs | fAuthenticated
}

func (f *flags) isDialbackKeyAuthorized() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.fs&fDialbackKeyAuthorized > 0
}

func (f *flags) setDialbackKeyAuthorized() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.fs = f.fs | fDialbackKeyAuthorized
}

func (f *flags) get() uint8 {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.fs
}
