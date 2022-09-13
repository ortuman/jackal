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
	"errors"
	"fmt"
	"strconv"

	"github.com/jackal-xmpp/stravaganza"
)

const (
	// RSMNamespace specifies XEP-0059 namespace constant value.
	RSMNamespace = "http://jabber.org/protocol/rsm"
)

var (
	// ErrPageNotFound will be returned by GetResultSetPage when page request cannot be satisfied.
	ErrPageNotFound = errors.New("page not found")
)

// Request represents a rsm request value.
type Request struct {
	After    string
	Before   string
	Index    int
	Max      int
	LastPage bool
}

// Result represents a rsm result value.
type Result struct {
	Index    int
	First    string
	Last     string
	Count    int
	Complete bool
}

// NewRequestFromElement returns a Request derived from an XML element.
func NewRequestFromElement(elem stravaganza.Element) (*Request, error) {
	var req Request
	var err error

	if n := elem.Name(); n != "set" {
		return nil, fmt.Errorf("xep0059: invalid set name: %s", n)
	}
	if ns := elem.Attribute(stravaganza.Namespace); ns != RSMNamespace {
		return nil, fmt.Errorf("xep0059: invalid set namespace: %s", ns)
	}
	if maxEl := elem.Child("max"); maxEl != nil {
		req.Max, err = strconv.Atoi(maxEl.Text())
		if err != nil {
			return nil, err
		}
	}
	if indexEl := elem.Child("index"); indexEl != nil {
		req.Index, err = strconv.Atoi(indexEl.Text())
		if err != nil {
			return nil, err
		}
	}
	if afterEl := elem.Child("after"); afterEl != nil {
		req.After = afterEl.Text()
	}
	if beforeEl := elem.Child("before"); beforeEl != nil {
		if beforeID := beforeEl.Text(); len(beforeID) > 0 {
			req.Before = beforeID
		} else {
			req.LastPage = true
		}
	}
	return &req, nil
}

// Element returns XML representation of a Result instance.
func (r *Result) Element() stravaganza.Element {
	sb := stravaganza.NewBuilder("set").
		WithAttribute(stravaganza.Namespace, RSMNamespace)

	if len(r.First) > 0 {
		sb.WithChild(
			stravaganza.NewBuilder("first").
				WithAttribute("index", strconv.Itoa(r.Index)).
				WithText(r.First).
				Build(),
		)
	}
	if len(r.Last) > 0 {
		sb.WithChild(
			stravaganza.NewBuilder("last").
				WithText(r.Last).
				Build(),
		)
	}
	sb.WithChild(
		stravaganza.NewBuilder("count").
			WithText(strconv.Itoa(r.Count)).
			Build(),
	)
	return sb.Build()
}

// GetResultSetPage returns result page based on the passed request.
func GetResultSetPage[T any](rs []T, req *Request, getID func(i T) string) ([]T, *Result, error) {
	var page []T
	var res *Result
	var err error

	switch {
	case len(rs) == 0 && req.Index == 0:
		return nil, &Result{Complete: true}, nil

	case req.LastPage:
		page, res, err = getPageByIndex(rs, lastIndex(len(rs), req.Max), req.Max)

	case req.Index > 0:
		page, res, err = getPageByIndex(rs, req.Index, req.Max)

	case len(req.After) > 0:
		page, res, err = getPageAfterID(rs, getID, req.After, req.Max)

	case len(req.Before) > 0:
		page, res, err = getPageBeforeID(rs, getID, req.Before, req.Max)

	case req.Max == 0:
		return nil, &Result{Count: len(rs)}, nil

	default:
		page, res, err = getPageByIndex(rs, 0, req.Max) // request first page
	}
	if err != nil {
		return nil, nil, err
	}
	res.First = getID(page[0])
	res.Last = getID(page[len(page)-1])

	return page, res, nil
}

func getPageByIndex[T any](rs []T, idx, max int) ([]T, *Result, error) {
	var page []T
	var res Result

	i := idx * max
	if i > len(rs)-1 {
		return nil, nil, ErrPageNotFound
	}

	lastIdx := len(rs) - 1
	for ; i < len(rs) && res.Count < max; i++ {
		if i >= lastIdx {
			res.Complete = true
		}
		page = append(page, rs[i])
		res.Count++
	}
	res.Index = idx

	return page, &res, nil
}

func getPageAfterID[T any](rs []T, getID func(i T) string, id string, max int) ([]T, *Result, error) {
	var page []T
	var res Result

	idIdx := getIDIndex(rs, getID, id)
	if idIdx == -1 {
		return nil, nil, ErrPageNotFound
	}
	startIdx := idIdx + 1

	lastIdx := len(rs) - 1
	for i := startIdx; i < len(rs) && res.Count < max; i++ {
		if i >= lastIdx {
			res.Complete = true
		}
		page = append(page, rs[i])
		res.Count++
	}
	res.Index = startIdx / max

	return page, &res, nil
}

func getPageBeforeID[T any](rs []T, getID func(i T) string, id string, max int) ([]T, *Result, error) {
	var page []T
	var res Result

	idIdx := getIDIndex(rs, getID, id)
	if idIdx == -1 {
		return nil, nil, ErrPageNotFound
	}
	startIdx := idIdx - max
	if startIdx < 0 {
		startIdx = 0
	}

	lastIdx := len(rs) - 1
	for i := startIdx; i < len(rs) && res.Count < max; i++ {
		if i >= lastIdx {
			res.Complete = true
		}
		page = append(page, rs[i])
		res.Count++
	}
	res.Index = startIdx / max

	return page, &res, nil
}

func getIDIndex[T any](rs []T, getID func(i T) string, id string) int {
	for i := 0; i < len(rs); i++ {
		if getID(rs[i]) != id {
			continue
		}
		return i
	}
	return -1
}

func lastIndex(len, max int) int {
	li := len/max - 1
	if len%max > 0 {
		li++
	}
	return li
}
