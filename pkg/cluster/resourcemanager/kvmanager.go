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

package resourcemanager

import (
	"context"
	"fmt"
	"strings"
	"sync"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	resourcemanagerpb "github.com/ortuman/jackal/pkg/c2s/pb"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	kvtypes "github.com/ortuman/jackal/pkg/cluster/kv/types"
	"github.com/ortuman/jackal/pkg/hook"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
)

const (
	resourceKeyPrefix = "r://"

	kvResourceManagerType = "kv"
)

type kvResources struct {
	mu    sync.RWMutex
	store map[string][]c2smodel.ResourceDesc
}

func (r *kvResources) get(username, resource string) c2smodel.ResourceDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rss := r.store[username]
	for _, res := range rss {
		if res.JID().Resource() != resource {
			continue
		}
		return res
	}
	return nil
}

func (r *kvResources) getAll(username string) []c2smodel.ResourceDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rss := r.store[username]
	if len(rss) == 0 {
		return nil
	}
	retVal := make([]c2smodel.ResourceDesc, len(rss))
	for i, res := range rss {
		retVal[i] = res
	}
	return retVal
}

func (r *kvResources) put(res c2smodel.ResourceDesc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	jd := res.JID()

	var username, resource = jd.Node(), jd.Resource()
	var found bool

	rss := r.store[username]
	for i := 0; i < len(rss); i++ {
		if rss[i].JID().Resource() != resource {
			continue
		}
		rss[i] = res
		found = true
		break
	}
	if !found {
		rss = append(rss, res)
	}
	r.store[username] = rss
	return
}

func (r *kvResources) del(username, resource string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rss := r.store[username]
	for i := 0; i < len(rss); i++ {
		if rss[i].JID().Resource() != resource {
			continue
		}
		rss = append(rss[:i], rss[i+1:]...)
		if len(rss) > 0 {
			r.store[username] = rss
		} else {
			delete(r.store, username)
		}
		return
	}
}

type kvManager struct {
	kv        kv.KV
	hk        *hook.Hooks
	logger    kitlog.Logger
	ctx       context.Context
	ctxCancel context.CancelFunc

	instResMu sync.RWMutex
	instRes   map[string]*kvResources

	stopCh chan struct{}
}

// NewKVManager creates a new resource manager given a KV storage instance.
func NewKVManager(kv kv.KV, hk *hook.Hooks, logger kitlog.Logger) Manager {
	ctx, ctxCancel := context.WithCancel(context.Background())
	return &kvManager{
		kv:        kv,
		hk:        hk,
		logger:    logger,
		ctx:       ctx,
		ctxCancel: ctxCancel,
		instRes:   make(map[string]*kvResources),
		stopCh:    make(chan struct{}),
	}
}

func (m *kvManager) PutResource(ctx context.Context, res c2smodel.ResourceDesc) error {
	b, err := resourceVal(res)
	if err != nil {
		return err
	}
	rKey := resourceKey(res.JID().Node(), res.JID().Resource())

	err = m.kv.Put(ctx, rKey, string(b))
	if err != nil {
		return err
	}
	m.inMemPut(res)
	return nil
}

func (m *kvManager) GetResource(_ context.Context, username, resource string) (c2smodel.ResourceDesc, error) {
	m.instResMu.RLock()
	defer m.instResMu.RUnlock()

	for _, kvr := range m.instRes {
		if res := kvr.get(username, resource); res != nil {
			return res, nil
		}
	}
	return nil, nil
}

func (m *kvManager) GetResources(_ context.Context, username string) ([]c2smodel.ResourceDesc, error) {
	m.instResMu.RLock()
	defer m.instResMu.RUnlock()

	var retVal []c2smodel.ResourceDesc
	for _, kvr := range m.instRes {
		retVal = append(retVal, kvr.getAll(username)...)
	}
	return retVal, nil
}

func (m *kvManager) DelResource(ctx context.Context, username, resource string) error {
	rKey := resourceKey(username, resource)

	if err := m.kv.Del(ctx, rKey); err != nil {
		return err
	}
	m.inMemDel(username, resource, instance.ID())
	return nil
}

func (m *kvManager) Start(ctx context.Context) error {
	m.hk.AddHook(hook.MemberListUpdated, m.onMemberListUpdated, hook.DefaultPriority)

	if err := m.watchKVResources(ctx); err != nil {
		return err
	}
	level.Info(m.logger).Log("msg", "started resource manager", "type", kvResourceManagerType)
	return nil
}

func (m *kvManager) Stop(_ context.Context) error {
	m.hk.RemoveHook(hook.MemberListUpdated, m.onMemberListUpdated)

	// stop watching changes...
	m.ctxCancel()
	<-m.stopCh

	level.Info(m.logger).Log("msg", "stopped resource manager", "type", kvResourceManagerType)
	return nil
}

func (m *kvManager) onMemberListUpdated(execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.MemberListInfo)
	if len(inf.UnregisteredKeys) == 0 {
		return nil
	}
	// drop unregistered instance(s) resources
	m.instResMu.Lock()
	for _, instanceID := range inf.UnregisteredKeys {
		delete(m.instRes, instanceID)
	}
	m.instResMu.Unlock()
	return nil
}

func (m *kvManager) watchKVResources(ctx context.Context) error {
	ch := make(chan error, 1)
	go func() {
		wCh := m.kv.Watch(m.ctx, resourceKeyPrefix, false)

		rss, err := m.getKVResources(ctx)
		if err != nil {
			ch <- err
			return
		}
		for _, res := range rss {
			m.inMemPut(res)
		}

		close(ch) // signal update

		// watch changes
		for wResp := range wCh {
			if err := wResp.Err; err != nil {
				level.Warn(m.logger).Log("msg", "error occurred watching resources", "err", err)
				continue
			}
			// process changes
			if err := m.processKVEvents(wResp.Events); err != nil {
				level.Warn(m.logger).Log("msg", "failed to process resources changes", "err", err)
			}
		}
		close(m.stopCh) // signal stop
	}()
	return <-ch
}

func (m *kvManager) getKVResources(ctx context.Context) ([]c2smodel.ResourceDesc, error) {
	vs, err := m.kv.GetPrefix(ctx, resourceKeyPrefix)
	if err != nil {
		return nil, err
	}
	return decodeKVResources(vs)
}

func (m *kvManager) processKVEvents(kvEvents []kvtypes.WatchEvent) error {
	for _, ev := range kvEvents {
		if isLocalKey(ev.Key) {
			continue // discard local changes
		}
		switch ev.Type {
		case kvtypes.Put:
			res, err := decodeResource(ev.Key, ev.Val)
			if err != nil {
				return err
			}
			m.inMemPut(res)

		case kvtypes.Del:
			username, resource, instanceID, err := extractKeyInfo(ev.Key)
			if err != nil {
				return err
			}
			m.inMemDel(username, resource, instanceID)
		}
	}
	return nil
}

func (m *kvManager) inMemPut(res c2smodel.ResourceDesc) {
	m.instResMu.Lock()
	defer m.instResMu.Unlock()

	instID := res.InstanceID()

	kvr := m.instRes[instID]
	if kvr == nil {
		kvr = &kvResources{
			store: make(map[string][]c2smodel.ResourceDesc),
		}
		m.instRes[instID] = kvr
	}
	kvr.put(res)
	return
}

func (m *kvManager) inMemDel(username, resource, instanceID string) {
	m.instResMu.RLock()
	defer m.instResMu.RUnlock()

	if kvr := m.instRes[instanceID]; kvr != nil {
		kvr.del(username, resource)
	}
}

func decodeKVResources(kvs map[string][]byte) ([]c2smodel.ResourceDesc, error) {
	var rs []c2smodel.ResourceDesc
	for k, v := range kvs {
		res, err := decodeResource(k, v)
		if err != nil {
			return nil, err
		}
		rs = append(rs, res)
	}
	return rs, nil
}

func decodeResource(key string, val []byte) (c2smodel.ResourceDesc, error) {
	errInvalidKeyFn := func(rKey string) error {
		return fmt.Errorf("invalid resource key format: %s", rKey)
	}

	ss0 := strings.Split(strings.TrimPrefix(key, resourceKeyPrefix), "@")
	if len(ss0) != 2 {
		return nil, errInvalidKeyFn(key)
	}

	var resInf resourcemanagerpb.ResourceInfo
	if err := proto.Unmarshal(val, &resInf); err != nil {
		return nil, err
	}
	ss1 := strings.Split(ss0[1], "/") // trim instance ID suffix
	if len(ss1) != 2 {
		return nil, errInvalidKeyFn(key)
	}
	username := ss0[0]
	resource := ss1[0]

	jd, _ := jid.New(username, resInf.Domain, resource, true)
	inf := c2smodel.NewInfoMapFromMap(resInf.Info)

	var pr *stravaganza.Presence
	if resInf.Presence != nil {
		var err error
		pr, err = stravaganza.NewBuilderFromProto(resInf.Presence).
			BuildPresence()
		if err != nil {
			return nil, err
		}
	}
	return c2smodel.NewResourceDesc(
		resInf.InstanceId,
		jd,
		pr,
		inf,
	), nil
}

func resourceKey(username, resource string) string {
	return fmt.Sprintf(
		"%s%s@%s/%s",
		resourceKeyPrefix,
		username,
		resource,
		instance.ID(),
	)
}

func resourceVal(res c2smodel.ResourceDesc) ([]byte, error) {
	var pbPresence *stravaganza.PBElement
	if res.Presence() != nil {
		pbPresence = res.Presence().Proto()
	}
	resInf := resourcemanagerpb.ResourceInfo{
		InstanceId: res.InstanceID(),
		Domain:     res.JID().Domain(),
		Info:       res.Info().Map(),
		Presence:   pbPresence,
	}
	return proto.Marshal(&resInf)
}

func isLocalKey(rKey string) bool {
	return strings.HasSuffix(rKey, fmt.Sprintf("/%s", instance.ID()))
}

func extractKeyInfo(rKey string) (username, resource, instanceID string, err error) {
	memberKey := strings.TrimPrefix(rKey, resourceKeyPrefix)
	ss := strings.Split(memberKey, "@")
	if len(ss) != 2 {
		return "", "", "", errInvalidKey(rKey)
	}
	username = ss[0]

	ss = strings.Split(ss[1], "/")
	if len(ss) != 2 {
		return "", "", "", errInvalidKey(rKey)
	}
	resource = ss[0]
	instanceID = ss[1]
	return
}

func errInvalidKey(rKey string) error {
	return fmt.Errorf("invalid kv resource key: %s", rKey)
}
