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

package xep0059

import (
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/stretchr/testify/require"
)

func TestRequest_NewFromElement(t *testing.T) {
	// given
	el := stravaganza.NewBuilder("set").
		WithAttribute(stravaganza.Namespace, RSMNamespace).
		WithChild(
			stravaganza.NewBuilder("max").
				WithText("10").
				Build(),
		).
		WithChild(
			stravaganza.NewBuilder("index").
				WithText("1").
				Build(),
		).
		WithChild(
			stravaganza.NewBuilder("after").
				WithText("peter@pixyland.org").
				Build(),
		).
		WithChild(
			stravaganza.NewBuilder("before").
				WithText("peter@rabbit.lit").
				Build(),
		).
		Build()

	// when
	req, err := NewRequestFromElement(el)

	// then
	require.NoError(t, err)

	require.Equal(t, 10, req.Max)
	require.Equal(t, 1, req.Index)
	require.Equal(t, "peter@pixyland.org", req.After)
	require.Equal(t, "peter@rabbit.lit", req.Before)
}

func TestResult_Element(t *testing.T) {
	// given
	r := Result{
		Index: 1,
		First: "f0",
		Last:  "l1",
		Count: 800,
	}

	// when
	el := r.Element()

	// then
	require.Equal(t, `<set xmlns='http://jabber.org/protocol/rsm'><first index='1'>f0</first><last>l1</last><count>800</count></set>`, el.String())
}

func Test_GetResultSetPage(t *testing.T) {
	tcs := map[string]struct {
		rs             []string
		req            Request
		expectedPage   []string
		expectedResult Result
		expectsError   bool
	}{
		"empty set": {
			req:            Request{Max: 10},
			expectedResult: Result{Count: 0, Complete: true},
		},
		"get page by index": {
			rs:             []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:            Request{Index: 2, Max: 3},
			expectedPage:   []string{"7", "8", "9"},
			expectedResult: Result{Index: 2, Count: 3, First: "7", Last: "9"},
		},
		"get out of bound index": {
			rs:           []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:          Request{Index: 4, Max: 3},
			expectsError: true,
		},
		"get last page": {
			rs:             []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:            Request{LastPage: true, Max: 3},
			expectedPage:   []string{"10"},
			expectedResult: Result{Index: 3, Count: 1, First: "10", Last: "10", Complete: true},
		},
		"get page after id": {
			rs:             []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:            Request{After: "3", Max: 4},
			expectedPage:   []string{"4", "5", "6", "7"},
			expectedResult: Result{Index: 0, Count: 4, First: "4", Last: "7"},
		},
		"get page after id - last page": {
			rs:             []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:            Request{After: "8", Max: 4},
			expectedPage:   []string{"9", "10"},
			expectedResult: Result{Index: 2, Count: 2, First: "9", Last: "10", Complete: true},
		},
		"get page after id - not found": {
			rs:           []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:          Request{After: "11", Max: 4},
			expectsError: true,
		},
		"get before id": {
			rs:             []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:            Request{Before: "9", Max: 2},
			expectedPage:   []string{"7", "8"},
			expectedResult: Result{Index: 3, Count: 2, First: "7", Last: "8"},
		},
		"get before id - first page": {
			rs:             []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:            Request{Before: "2", Max: 4},
			expectedPage:   []string{"1", "2", "3", "4"},
			expectedResult: Result{Index: 0, Count: 4, First: "1", Last: "4"},
		},
		"get before id - not found": {
			rs:           []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:          Request{Before: "11", Max: 4},
			expectsError: true,
		},
		"get results count": {
			rs:             []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			req:            Request{Max: 0},
			expectedResult: Result{Count: 10},
		},
	}
	for tName, tc := range tcs {
		t.Run(tName, func(t *testing.T) {
			page, res, err := GetResultSetPage(tc.rs, &tc.req, func(s string) string { return s })
			if tc.expectsError {
				require.Error(t, err)
			} else {
				require.Equal(t, &tc.expectedResult, res)
				require.Equal(t, tc.expectedPage, page)
			}
		})
	}
}
