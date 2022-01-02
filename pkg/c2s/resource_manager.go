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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ortuman/jackal/pkg/cluster/instance"

	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	resourcemanagerpb "github.com/ortuman/jackal/pkg/c2s/pb"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/log"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
)

const (
	resourceKeyPrefix = "r://"
)

const clearActiveKeyTimeout = time.Minute

// ResourceManager type is in charge of keeping track of all cluster resources.
type ResourceManager struct {
	kv        kv.KV
	ctx       context.Context
	ctxCancel context.CancelFunc

	storeMu sync.RWMutex
	store   map[string][]c2smodel.ResourceDesc

	// active put key set
	stopCh chan struct{}
}

// NewResourceManager creates a new resource manager given a KV storage instance.
func NewResourceManager(kv kv.KV) *ResourceManager {
	ctx, ctxCancel := context.WithCancel(context.Background())
	return &ResourceManager{
		kv:        kv,
		ctx:       ctx,
		ctxCancel: ctxCancel,
		store:     make(map[string][]c2smodel.ResourceDesc),
		stopCh:    make(chan struct{}),
	}
}

// PutResource registers or updates a resource into the manager.
func (m *ResourceManager) PutResource(ctx context.Context, res c2smodel.ResourceDesc) error {
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

// GetResource returns a previously registered resource.
func (m *ResourceManager) GetResource(_ context.Context, username, resource string) (c2smodel.ResourceDesc, error) {
	m.storeMu.RLock()
	defer m.storeMu.RUnlock()

	rss := m.store[username]
	for _, res := range rss {
		if res.JID().Resource() != resource {
			continue
		}
		return res, nil
	}
	return nil, nil
}

// GetResources returns all user registered resources.
func (m *ResourceManager) GetResources(_ context.Context, username string) ([]c2smodel.ResourceDesc, error) {
	m.storeMu.RLock()
	defer m.storeMu.RUnlock()

	rss := m.store[username]
	if len(rss) == 0 {
		return nil, nil
	}
	retVal := make([]c2smodel.ResourceDesc, len(rss))
	for i, res := range rss {
		retVal[i] = res
	}
	return retVal, nil
}

// DelResource removes a registered resource from the manager.
func (m *ResourceManager) DelResource(ctx context.Context, username, resource string) error {
	rKey := resourceKey(username, resource)

	if err := m.kv.Del(ctx, rKey); err != nil {
		return err
	}
	m.inMemDel(username, resource)
	return nil
}

// Start starts resource manager.
func (m *ResourceManager) Start(ctx context.Context) error {
	if err := m.watchKVResources(ctx); err != nil {
		return err
	}
	log.Infow("started C2S resource manager")
	return nil
}

// Stop stops resource manager.
func (m *ResourceManager) Stop(_ context.Context) error {
	// stop watching changes...
	m.ctxCancel()
	<-m.stopCh

	log.Infow("stopped C2S resource manager")
	return nil
}

func (m *ResourceManager) watchKVResources(ctx context.Context) error {
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
				log.Warnf("Error occurred watching resources: %v", err)
				continue
			}
			// process changes
			if err := m.processKVEvents(wResp.Events); err != nil {
				log.Warnf("Failed to process resources changes: %v", err)
			}
		}
		close(m.stopCh) // signal stop
	}()
	return <-ch
}

func (m *ResourceManager) getKVResources(ctx context.Context) ([]c2smodel.ResourceDesc, error) {
	vs, err := m.kv.GetPrefix(ctx, resourceKeyPrefix)
	if err != nil {
		return nil, err
	}
	return decodeKVResources(vs)
}

func (m *ResourceManager) processKVEvents(kvEvents []kv.WatchEvent) error {
	for _, ev := range kvEvents {
		if isLocalKey(ev.Key) {
			continue // discard local changes
		}
		switch ev.Type {
		case kv.Put:
			res, err := decodeResource(ev.Key, ev.Val)
			if err != nil {
				return err
			}
			m.inMemPut(res)

		case kv.Del:
			memberKey := strings.TrimPrefix(ev.Key, resourceKeyPrefix)
			ss := strings.Split(memberKey, "@")
			if len(ss) != 2 {
				return fmt.Errorf("invalid kv resource key: %s", ev.Key)
			}
			var username, resource = ss[0], ss[1]

			m.inMemDel(username, resource)
		}
	}
	return nil
}

func (m *ResourceManager) inMemPut(res c2smodel.ResourceDesc) {
	m.storeMu.Lock()
	defer m.storeMu.Unlock()

	jd := res.JID()

	var username, resource = jd.Node(), jd.Resource()
	var found bool

	rss := m.store[username]
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
	m.store[username] = rss
	return
}

func (m *ResourceManager) inMemDel(username, resource string) {
	m.storeMu.Lock()
	defer m.storeMu.Unlock()

	rss := m.store[username]
	for i := 0; i < len(rss); i++ {
		if rss[i].JID().Resource() != resource {
			continue
		}
		rss = append(rss[:i], rss[i+1:]...)
		if len(rss) > 0 {
			m.store[username] = rss
		} else {
			delete(m.store, username)
		}
		return
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
	inf := c2smodel.Info{M: resInf.Info}

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
		Info:       res.Info().M,
		Presence:   pbPresence,
	}
	return proto.Marshal(&resInf)
}

func isLocalKey(rKey string) bool {
	return strings.HasSuffix(rKey, fmt.Sprintf("/%s", instance.ID()))
}
