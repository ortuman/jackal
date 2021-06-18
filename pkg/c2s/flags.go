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

package c2s

import "sync"

const (
	fSecured        uint8 = 1 << 0
	fAuthenticated        = 1 << 1
	fCompressed           = 1 << 2
	fBinded               = 1 << 3
	fSessionStarted       = 1 << 4
)

type flags struct {
	mtx sync.RWMutex
	flg uint8
}

func (f *flags) isSecured() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fSecured > 0
}

func (f *flags) setSecured() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fSecured
}

func (f *flags) isAuthenticated() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fAuthenticated > 0
}

func (f *flags) setAuthenticated() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fAuthenticated
}

func (f *flags) isCompressed() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fCompressed > 0
}

func (f *flags) setCompressed() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fCompressed
}

func (f *flags) isBinded() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fBinded > 0
}

func (f *flags) setBinded() {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	f.flg = f.flg | fBinded
}

func (f *flags) isSessionStarted() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fSessionStarted > 0
}

func (f *flags) setSessionStarted() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fSessionStarted
}
