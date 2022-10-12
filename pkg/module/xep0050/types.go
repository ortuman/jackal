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

package xep0050

import (
	"context"

	"github.com/jackal-xmpp/stravaganza/jid"

	"github.com/ortuman/jackal/pkg/module/xep0004"
)

// Action represents an ad-hoc command action.
type Action int

const (
	ActionUnknown Action = iota
	ActionExecute
	ActionCancel
	ActionPrev
	ActionNext
	ActionComplete
)

// String returns the string representation of an ad-hoc command action.
func (a Action) String() string {
	switch a {
	case ActionExecute:
		return "execute"
	case ActionCancel:
		return "cancel"
	case ActionPrev:
		return "prev"
	case ActionNext:
		return "next"
	case ActionComplete:
		return "complete"
	}
	return ""
}

// Status represents an ad-hoc command status.
type Status int

const (
	StatusExecuting Status = iota
	StatusCanceled
	StatusCompleted
)

// String returns the string representation of an ad-hoc command status.
func (s Status) String() string {
	switch s {
	case StatusExecuting:
		return "executing"
	case StatusCanceled:
		return "canceled"
	case StatusCompleted:
		return "completed"
	}
	return ""
}

// Session represents an ad-hoc command session.
type Session struct {
	ID            string
	Node          string
	Owner         string
	Status        Status
	Stage         int
	Actions       []Action
	ExecuteAction Action
	Data          map[string]interface{}
}

// AdHocCommand represents an ad-hoc command.
type AdHocCommand interface {
	// Node returns the command node.
	Node() string

	// Name returns the command name.
	Name() string

	// IsAllowed returns true if the requester is allowed to execute this command.
	IsAllowed(jid *jid.JID) bool

	// Execute executes the command.
	Execute(ctx context.Context, session *Session, action Action, form *xep0004.DataForm) (*xep0004.DataForm, error)
}
