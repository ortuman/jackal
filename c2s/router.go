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

import (
	"context"
	"sort"

	"github.com/ortuman/jackal/event"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/cluster/instance"
	clusterrouter "github.com/ortuman/jackal/cluster/router"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/stream"
)

type c2sRouter struct {
	local   localRouter
	cluster clusterRouter
	resMng  resourceManager
	rep     repository.Repository
	sn      *sonar.Sonar
}

// NewRouter creates and returns an initialized C2S router.
func NewRouter(
	localRouter *LocalRouter,
	clusterRouter *clusterrouter.Router,
	resMng *ResourceManager,
	rep repository.Repository,
	sn *sonar.Sonar,
) router.C2SRouter {
	return &c2sRouter{
		local:   localRouter,
		cluster: clusterRouter,
		resMng:  resMng,
		rep:     rep,
		sn:      sn,
	}
}

func (r *c2sRouter) Route(ctx context.Context, stanza stravaganza.Stanza, routingOpts router.RoutingOptions) error {
	fromJID := stanza.FromJID()
	toJID := stanza.ToJID()

	// apply validations
	username := stanza.ToJID().Node()
	if (routingOpts & router.CheckUserExistence) > 0 {
		exists, err := r.rep.UserExists(ctx, username) // user exists?
		if err != nil {
			return err
		}
		if !exists {
			return router.ErrNotExistingAccount
		}
	}
	if (routingOpts & router.ValidateSenderJID) > 0 {
		if r.isBlockedJID(ctx, fromJID, toJID.Node()) { // check whether sender JID is blocked
			return router.ErrBlockedSender
		}
	}
	// get user available resources
	rss, err := r.resMng.GetResources(ctx, username)
	if err != nil {
		return err
	}
	return r.route(ctx, stanza, rss)
}

func (r *c2sRouter) Disconnect(ctx context.Context, res *model.Resource, streamErr *streamerror.Error) error {
	if instance.ID() == res.InstanceID {
		return r.local.Disconnect(res.JID.Node(), res.JID.Resource(), streamErr)
	}
	return r.cluster.Disconnect(ctx, res.JID.Node(), res.JID.Resource(), streamErr, res.InstanceID)
}

func (r *c2sRouter) Register(stm stream.C2S) error {
	if err := r.local.Register(stm); err != nil {
		return err
	}
	log.Infow("Registered C2S stream", "id", stm.ID())
	return nil
}

func (r *c2sRouter) Bind(id stream.C2SID) error {
	_, err := r.local.Bind(id)
	if err != nil {
		return err
	}
	return nil
}

func (r *c2sRouter) Unregister(stm stream.C2S) error {
	if err := r.local.Unregister(stm); err != nil {
		return err
	}
	log.Infow("Unregistered C2S stream", "id", stm.ID())
	return nil
}

func (r *c2sRouter) LocalStream(username, resource string) stream.C2S {
	return r.local.Stream(username, resource)
}

func (r *c2sRouter) Start(ctx context.Context) error {
	if err := r.cluster.Start(ctx); err != nil {
		return err
	}
	return r.local.Start(ctx)
}

func (r *c2sRouter) Stop(ctx context.Context) error {
	if err := r.cluster.Stop(ctx); err != nil {
		return err
	}
	return r.local.Stop(ctx)
}

func (r *c2sRouter) route(ctx context.Context, stanza stravaganza.Stanza, resources []model.Resource) error {
	if len(resources) == 0 {
		return router.ErrUserNotAvailable
	}
	var targets []jid.JID

	toJID := stanza.ToJID()
	if toJID.IsFullWithUser() {
		// route to full resource
		for _, res := range resources {
			if res.JID.Resource() != toJID.Resource() {
				continue
			}
			if err := r.routeTo(ctx, stanza, &res); err != nil {
				return err
			}
			targets = append(targets, *res.JID)
			goto postRoutedEvent
		}
		return router.ErrResourceNotFound
	}
	switch stanza.(type) {
	case *stravaganza.Message:
		// route to highest priority resources
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].Priority() > resources[j].Priority()
		})
		p0 := resources[0].Priority() // highest priority

		var routed bool
		for _, res := range resources {
			if res.Priority() < 0 || res.Priority() != p0 {
				break
			}
			if err := r.routeTo(ctx, stanza, &res); err != nil {
				return err
			}
			targets = append(targets, *res.JID)
			routed = true
		}
		if !routed {
			return router.ErrUserNotAvailable
		}
		goto postRoutedEvent
	}
	// broadcast to all resources
	for _, res := range resources {
		if err := r.routeTo(ctx, stanza, &res); err != nil {
			return err
		}
		targets = append(targets, *res.JID)
	}

postRoutedEvent:
	return r.sn.Post(ctx, sonar.NewEventBuilder(event.C2SRouterStanzaRouted).
		WithInfo(&event.C2SRouterEventInfo{
			Targets: targets,
			Stanza:  stanza,
		}).
		Build(),
	)
}

func (r *c2sRouter) routeTo(ctx context.Context, stanza stravaganza.Stanza, toRes *model.Resource) error {
	if toRes.InstanceID == instance.ID() {
		return r.local.Route(stanza, toRes.JID.Node(), toRes.JID.Resource())
	}
	return r.cluster.Route(ctx, stanza, toRes.JID.Node(), toRes.JID.Resource(), toRes.InstanceID)
}

func (r *c2sRouter) isBlockedJID(ctx context.Context, destJID *jid.JID, username string) bool {
	blockList, err := r.rep.FetchBlockListItems(ctx, username)
	if err != nil {
		log.Errorf("Failed to fetch block list items: %v", err)
		return false
	}
	if len(blockList) == 0 {
		return false
	}
	blockListJIDs := make([]jid.JID, len(blockList))
	for i, listItem := range blockList {
		j, _ := jid.NewWithString(listItem.JID, true)
		blockListJIDs[i] = *j
	}
	for _, blockedJID := range blockListJIDs {
		if blockedJID.Matches(destJID) {
			return true
		}
	}
	return false
}
