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

package s2s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOutProvider_GetOut(t *testing.T) {
	// given
	op := &OutProvider{
		outStreams: make(map[string]s2sOut),
	}
	var out *s2sOutMock
	op.newOutFn = func(sender, target string) s2sOut {
		out = &s2sOutMock{}
		out.dialFunc = func(ctx context.Context) error { return nil }
		out.startFunc = func() error { return nil }
		return out
	}

	// when
	conn1, _ := op.GetOut(context.Background(), "jackal.im", "jabber.org")
	conn2, _ := op.GetOut(context.Background(), "jackal.im", "jabber.org")

	time.Sleep(time.Second) // wait until started

	// then
	require.Equal(t, conn1, conn2)

	require.Len(t, conn1.(*s2sOutMock).startCalls(), 1)
	require.Len(t, conn1.(*s2sOutMock).dialCalls(), 1)
}

func TestOutProvider_GetDialback(t *testing.T) {
	// given
	op := &OutProvider{
		outStreams: make(map[string]s2sOut),
	}
	op.newDbFn = func(sender, target string, dbParam DialbackParams) s2sDialback {
		db := &s2sDialbackMock{}
		db.dialFunc = func(ctx context.Context) error { return nil }
		db.startFunc = func() error { return nil }
		return db
	}

	// when
	conn1, _ := op.GetDialback(context.Background(), "jackal.im", "jabber.org", DialbackParams{})
	conn2, _ := op.GetDialback(context.Background(), "jackal.im", "jabber.org", DialbackParams{})

	time.Sleep(time.Second) // wait until started

	// then
	require.NotEqual(t, conn1, conn2)

	require.Len(t, conn1.(*s2sDialbackMock).startCalls(), 1)
	require.Len(t, conn1.(*s2sDialbackMock).dialCalls(), 1)

	require.Len(t, conn2.(*s2sDialbackMock).startCalls(), 1)
	require.Len(t, conn2.(*s2sDialbackMock).dialCalls(), 1)
}
