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
	"sync"
	"time"

	"github.com/ortuman/jackal/pkg/module/xep0004"

	"github.com/google/uuid"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"

	"github.com/jackal-xmpp/stravaganza"
)

// CommandNamespace represents the ad-hoc command namespace.
const CommandNamespace = "http://jabber.org/protocol/commands"

type Config struct {
	MaxSessionsPerNode int
	SessionTTL         time.Duration
}

type Handler struct {
	cfg      Config
	router   router.Router
	commands map[string]AdHocCommand

	mu             sync.RWMutex
	activeSessions map[string]*userSessions
	timers         map[string]*time.Timer
}

type userSessions struct {
	sync.Mutex
	sessions []*Session
}

// NewCommandHandler returns a new initialized ad-hoc command handler.
func NewCommandHandler(cfg Config, router router.Router) *Handler {
	return &Handler{
		cfg:            cfg,
		router:         router,
		commands:       make(map[string]AdHocCommand),
		activeSessions: make(map[string]*userSessions),
		timers:         make(map[string]*time.Timer),
	}
}

// RegisterCommand registers a new ad-hoc command.
func (m *Handler) RegisterCommand(cmd AdHocCommand) {
	m.mu.Lock()
	defer m.mu.Unlock()

	node := cmd.Node()

	if _, ok := m.commands[node]; ok {
		return // already registered
	}
	m.commands[node] = cmd
}

func (m *Handler) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	cmdElement := iq.ChildNamespace("command", CommandNamespace)
	if cmdElement == nil || !iq.IsSet() {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	node := cmdElement.Attribute("node")

	m.mu.RLock()
	cmd := m.commands[node]
	m.mu.RUnlock()

	if cmd == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.ItemNotFound))
		return nil
	}
	fromJID := iq.FromJID()

	if !cmd.IsAllowed(fromJID) {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil
	}
	action := getCommandAction(cmdElement)
	if action == ActionUnknown {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	requester := fromJID.ToBareJID().String()

	m.mu.RLock()
	requesterSessions := m.activeSessions[requester]
	m.mu.RUnlock()

	if requesterSessions == nil {
		m.mu.Lock()
		requesterSessions = m.activeSessions[requester]
		if requesterSessions == nil {
			requesterSessions = &userSessions{}
			m.activeSessions[requester] = requesterSessions
		}
		m.mu.Unlock()
	}
	var session *Session

	sessionID := cmdElement.Attribute("sessionid")

	requesterSessions.Lock()
	switch {
	case action == ActionExecute && len(sessionID) == 0:
		// check if requester initiated max allowed sessions
		var activeNodeSessions int
		for _, s := range requesterSessions.sessions {
			if s.Node == node {
				activeNodeSessions++
			}
		}
		if activeNodeSessions > m.cfg.MaxSessionsPerNode {
			requesterSessions.Unlock()
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.NotAllowed))
			return nil
		}
		session = m.registerSession(requester, node)
		requesterSessions.sessions = append(requesterSessions.sessions, session)

	default:
		for _, ss := range requesterSessions.sessions {
			if ss.ID == sessionID {
				session = ss
				break
			}
		}

		if session == nil {
			requesterSessions.Unlock()
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.NotAllowed))
			return nil
		}
	}
	requesterSessions.Unlock()

	// extract requester data form
	var reqForm *xep0004.DataForm

	if x := cmdElement.ChildNamespace("x", xep0004.FormNamespace); x != nil {
		var err error
		reqForm, err = xep0004.NewFormFromElement(x)
		if err != nil {
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
			return nil
		}
	}

	// execute command
	resForm, err := cmd.Execute(ctx, session, action, reqForm)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return nil
	}

	// send result
	respBuilder := stravaganza.NewBuilder("command").
		WithAttribute("xmlns", CommandNamespace).
		WithAttribute("node", node).
		WithAttribute("sessionid", session.ID).
		WithAttribute("status", session.Status.String())

	switch session.Status {
	case StatusExecuting:
		actionsBuilder := stravaganza.NewBuilder("actions").
			WithAttribute("execute", session.ExecuteAction.String())

		for _, action := range session.Actions {
			actionsBuilder.WithChild(
				stravaganza.NewBuilder(action.String()).
					Build(),
			)
		}
		respBuilder.WithChild(actionsBuilder.Build())

	case StatusCompleted, StatusCanceled:
		m.unregisterSession(session)
	}

	if resForm != nil {
		respBuilder.WithChild(resForm.Element())
	}
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, respBuilder.Build()))
	return nil
}

func (m *Handler) registerSession(owner, node string) *Session {
	ss := &Session{
		ID:     uuid.New().String(),
		Node:   node,
		Owner:  owner,
		Status: StatusExecuting,
		Data:   make(map[string]interface{}),
	}
	tm := time.AfterFunc(m.cfg.SessionTTL, func() {
		m.unregisterSession(ss)
	})
	m.mu.Lock()
	m.timers[ss.ID] = tm
	m.mu.Unlock()
	return ss
}

func (m *Handler) unregisterSession(ss *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// clear timer
	tm := m.timers[ss.ID]
	if tm != nil {
		tm.Stop()
		delete(m.timers, ss.ID)
	}

	panic("remove session from activeSessions")
}

// IsCommandIQ returns whether a given IQ is an ad-hoc command.
func IsCommandIQ(iq *stravaganza.IQ) bool {
	return iq.ChildNamespace("command", CommandNamespace) != nil
}

func getCommandAction(cmdElement stravaganza.Element) Action {
	switch cmdElement.Attribute("action") {
	case "cancel":
		return ActionCancel
	case "complete":
		return ActionComplete
	case "execute":
		return ActionExecute
	case "next":
		return ActionNext
	case "prev":
		return ActionPrev
	case "":
		return ActionExecute
	default:
		return ActionUnknown
	}
}
