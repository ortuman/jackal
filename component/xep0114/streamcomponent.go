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

package xep0114

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"
)

type streamComponent struct {
	stm *inComponent
}

func (sc *streamComponent) Host() string { return sc.stm.getJID().Domain() }
func (sc *streamComponent) Name() string { return "" }

func (sc *streamComponent) ProcessStanza(_ context.Context, stanza stravaganza.Stanza) error {
	_ = sc.stm.sendStanza(stanza)
	return nil
}

func (sc *streamComponent) Start(_ context.Context) error { return nil }
func (sc *streamComponent) Stop(_ context.Context) error  { return nil }
