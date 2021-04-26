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
	fBounded              = 1 << 3
	fSessionStarted       = 1 << 4
)

type inC2SFlags struct {
	mtx sync.RWMutex
	flg uint8
}

func (f *inC2SFlags) isSecured() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fSecured > 0
}

func (f *inC2SFlags) setSecured() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fSecured
}

func (f *inC2SFlags) isAuthenticated() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fAuthenticated > 0
}

func (f *inC2SFlags) setAuthenticated() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fAuthenticated
}

func (f *inC2SFlags) isCompressed() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fCompressed > 0
}

func (f *inC2SFlags) setCompressed() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fCompressed
}

func (f *inC2SFlags) isBounded() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fBounded > 0
}

func (f *inC2SFlags) setBounded() {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	f.flg = f.flg | fBounded
}

func (f *inC2SFlags) isSessionStarted() bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.flg&fSessionStarted > 0
}

func (f *inC2SFlags) setSessionStarted() {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.flg = f.flg | fSessionStarted
}
