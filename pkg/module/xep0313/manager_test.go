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

package xep0313

import (
	"testing"
	"time"

	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	"github.com/ortuman/jackal/pkg/module/xep0004"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestManager_FormToFields(t *testing.T) {
	tcs := map[string]struct {
		form    *xep0004.DataForm
		filters *archivemodel.Filters
	}{
		"by jid": {
			form: &xep0004.DataForm{
				Type: xep0004.Submit,
				Fields: []xep0004.Field{
					{Var: xep0004.FormType, Type: xep0004.Hidden, Values: []string{mamNamespace}},
					{Var: "with", Values: []string{"juliet@capulet.lit"}},
				},
			},
			filters: &archivemodel.Filters{
				With: "juliet@capulet.lit",
			},
		},
		"time received": {
			form: &xep0004.DataForm{
				Type: xep0004.Submit,
				Fields: []xep0004.Field{
					{Var: xep0004.FormType, Type: xep0004.Hidden, Values: []string{mamNamespace}},
					{Var: "start", Values: []string{"2010-06-07T00:00:00Z"}},
					{Var: "end", Values: []string{"2010-07-07T13:23:54Z"}},
				},
			},
			filters: &archivemodel.Filters{
				Start: timestamppb.New(time.Date(2010, 06, 07, 00, 00, 00, 00, time.UTC)),
				End:   timestamppb.New(time.Date(2010, 07, 07, 13, 23, 54, 00, time.UTC)),
			},
		},
		"after/before id": {
			form: &xep0004.DataForm{
				Type: xep0004.Submit,
				Fields: []xep0004.Field{
					{Var: xep0004.FormType, Type: xep0004.Hidden, Values: []string{mamNamespace}},
					{Var: "after-id", Values: []string{"28482-98726-73623"}},
					{Var: "before-id", Values: []string{"09af3-cc343-b409f"}},
				},
			},
			filters: &archivemodel.Filters{
				AfterId:  "28482-98726-73623",
				BeforeId: "09af3-cc343-b409f",
			},
		},
		"ids": {
			form: &xep0004.DataForm{
				Type: xep0004.Submit,
				Fields: []xep0004.Field{
					{Var: xep0004.FormType, Type: xep0004.Hidden, Values: []string{mamNamespace}},
					{Var: "ids", Values: []string{"28482-98726-73623", "09af3-cc343-b409f"}},
				},
			},
			filters: &archivemodel.Filters{
				Ids: []string{"28482-98726-73623", "09af3-cc343-b409f"},
			},
		},
	}
	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			filters, err := formToFilters(tc.form)

			require.NoError(t, err)
			require.Equal(t, tc.filters.String(), filters.String())
		})
	}
}
