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

	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	resourcemanagerpb "github.com/ortuman/jackal/c2s/pb"
	"github.com/ortuman/jackal/cluster/kv"
	coremodel "github.com/ortuman/jackal/model/core"
)

const (
	resourceKeyPrefix = "r://"
)

// ResourceManager type is in charge of keeping track of all cluster resources.
type ResourceManager struct {
	kv kv.KV
}

// NewResourceManager creates a new resource manager given a KV storage instance.
func NewResourceManager(kv kv.KV) *ResourceManager {
	return &ResourceManager{kv: kv}
}

// PutResource registers or updates a resource into the manager.
func (m *ResourceManager) PutResource(ctx context.Context, res *coremodel.Resource) error {
	b, err := resourceVal(res)
	if err != nil {
		return err
	}
	return m.kv.Put(
		ctx,
		resourceKey(res.JID.Node(), res.JID.Resource()),
		string(b),
	)
}

// GetResource returns a previously registered resource.
func (m *ResourceManager) GetResource(ctx context.Context, username, resource string) (*coremodel.Resource, error) {
	kvs, err := m.kv.GetPrefix(ctx, fmt.Sprintf("%s%s@%s", resourceKeyPrefix, username, resource))
	if err != nil {
		return nil, err
	}
	rs, err := m.deserializeKVResources(kvs)
	if err != nil {
		return nil, err
	}
	if len(kvs) == 0 {
		return nil, nil
	}
	return &rs[0], nil
}

// GetResources returns all user registered resources.
func (m *ResourceManager) GetResources(ctx context.Context, username string) ([]coremodel.Resource, error) {
	kvs, err := m.kv.GetPrefix(ctx, fmt.Sprintf("%s%s", resourceKeyPrefix, username))
	if err != nil {
		return nil, err
	}
	return m.deserializeKVResources(kvs)
}

// DelResource removes a registered resource from the manager.
func (m *ResourceManager) DelResource(ctx context.Context, username, resource string) error {
	return m.kv.Del(ctx, resourceKey(username, resource))
}

func (m *ResourceManager) deserializeKVResources(kvs map[string][]byte) ([]coremodel.Resource, error) {
	var rs []coremodel.Resource
	for k, v := range kvs {
		res, err := decodeResource(k, v)
		if err != nil {
			return nil, err
		}
		rs = append(rs, *res)
	}
	return rs, nil
}

func decodeResource(key string, val []byte) (*coremodel.Resource, error) {
	var res coremodel.Resource

	ss := strings.Split(strings.TrimPrefix(key, resourceKeyPrefix), "@")
	if len(ss) != 2 {
		return nil, fmt.Errorf("resourcemanager: invalid key format: %s", key)
	}

	var resInf resourcemanagerpb.ResourceInfo
	if err := proto.Unmarshal(val, &resInf); err != nil {
		return nil, err
	}
	res.InstanceID = resInf.InstanceId
	res.JID, _ = jid.New(ss[0], resInf.Domain, ss[1], true)
	res.Context = resInf.Context

	if resInf.Presence != nil {
		pr, err := stravaganza.NewBuilderFromProto(resInf.Presence).
			BuildPresence(false)
		if err != nil {
			return nil, err
		}
		res.Presence = pr
	}
	return &res, nil
}

func resourceKey(username, resource string) string {
	return fmt.Sprintf(
		"%s%s@%s",
		resourceKeyPrefix,
		username,
		resource,
	)
}

func resourceVal(res *coremodel.Resource) ([]byte, error) {
	var pbPresence *stravaganza.PBElement
	if res.Presence != nil {
		pbPresence = res.Presence.Proto()
	}
	resInf := resourcemanagerpb.ResourceInfo{
		InstanceId: res.InstanceID,
		Domain:     res.JID.Domain(),
		Context:    res.Context,
		Presence:   pbPresence,
	}
	return proto.Marshal(&resInf)
}
