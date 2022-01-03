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

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"

	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	clusterrouter "github.com/ortuman/jackal/pkg/cluster/router"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type c2sRouter struct {
	local   localRouter
	cluster clusterRouter
	resMng  resourceManager
	rep     repository.Repository
	hk      *hook.Hooks
	logger  kitlog.Logger
}

// NewRouter creates and returns an initialized C2S router.
func NewRouter(
	localRouter *LocalRouter,
	clusterRouter *clusterrouter.Router,
	resMng *ResourceManager,
	rep repository.Repository,
	hk *hook.Hooks,
	logger kitlog.Logger,
) router.C2SRouter {
	return &c2sRouter{
		local:   localRouter,
		cluster: clusterRouter,
		resMng:  resMng,
		rep:     rep,
		hk:      hk,
		logger:  logger,
	}
}

func (r *c2sRouter) Route(ctx context.Context, stanza stravaganza.Stanza, routingOpts router.RoutingOptions) (targets []jid.JID, err error) {
	// apply validations
	username := stanza.ToJID().Node()
	if (routingOpts & router.CheckUserExistence) > 0 {
		exists, err := r.rep.UserExists(ctx, username) // user exists?
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, router.ErrNotExistingAccount
		}
	}
	// get user available resources
	rss, err := r.resMng.GetResources(ctx, username)
	if err != nil {
		return nil, err
	}
	return r.route(ctx, stanza, rss)
}

func (r *c2sRouter) Disconnect(ctx context.Context, res c2smodel.ResourceDesc, streamErr *streamerror.Error) error {
	var username, resource = res.JID().Node(), res.JID().Resource()

	if instance.ID() == res.InstanceID() {
		return r.local.Disconnect(username, resource, streamErr)
	}
	return r.cluster.Disconnect(ctx, username, resource, streamErr, res.InstanceID())
}

func (r *c2sRouter) Register(stm stream.C2S) error {
	if err := r.local.Register(stm); err != nil {
		return err
	}
	level.Info(r.logger).Log("msg", "registered C2S stream", "id", stm.ID())
	return nil
}

func (r *c2sRouter) Bind(id stream.C2SID) error {
	stm, err := r.local.Bind(id)
	if err != nil {
		return err
	}
	level.Info(r.logger).Log("msg", "binded C2S stream", "id", id,
		"username", stm.Username(),
		"resource", stm.Resource())
	return nil
}

func (r *c2sRouter) Unregister(stm stream.C2S) error {
	if err := r.local.Unregister(stm); err != nil {
		return err
	}
	level.Info(r.logger).Log("msg", "unregistered C2S stream", "id", stm.ID())
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

func (r *c2sRouter) route(ctx context.Context, stanza stravaganza.Stanza, resources []c2smodel.ResourceDesc) ([]jid.JID, error) {
	if len(resources) == 0 {
		return nil, router.ErrUserNotAvailable
	}
	var targets []jid.JID

	toJID := stanza.ToJID()
	if toJID.IsFullWithUser() {
		// route to full resource
		for _, res := range resources {
			if res.JID().Resource() != toJID.Resource() {
				continue
			}
			if err := r.routeTo(ctx, stanza, res); err != nil {
				return nil, err
			}
			return []jid.JID{*res.JID()}, nil
		}
		return nil, router.ErrResourceNotFound
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
			if err := r.routeTo(ctx, stanza, res); err != nil {
				return nil, err
			}
			targets = append(targets, *res.JID())
			routed = true
		}
		if !routed {
			return nil, router.ErrUserNotAvailable
		}
		return targets, nil
	}
	// broadcast to all resources
	for _, res := range resources {
		if err := r.routeTo(ctx, stanza, res); err != nil {
			return nil, err
		}
		targets = append(targets, *res.JID())
	}
	return targets, nil
}

func (r *c2sRouter) routeTo(ctx context.Context, stanza stravaganza.Stanza, toRes c2smodel.ResourceDesc) error {
	var username, resource = toRes.JID().Node(), toRes.JID().Resource()

	if toRes.InstanceID() == instance.ID() {
		return r.local.Route(stanza, username, resource)
	}
	return r.cluster.Route(ctx, stanza, username, resource, toRes.InstanceID())
}
