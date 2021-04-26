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

package memberlist

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/event"
	"github.com/ortuman/jackal/pkg/log"
	coremodel "github.com/ortuman/jackal/pkg/model/core"
	"github.com/ortuman/jackal/pkg/version"
)

const (
	memberKeyPrefix   = "i://"
	memberValueFormat = "a=%s cv=%s"
)

var (
	interfaceAddrs = net.InterfaceAddrs
)

// MemberList keeps and manages cluster memberlist set.
type MemberList struct {
	localPort int
	kv        kv.KV
	ctx       context.Context
	ctxCancel context.CancelFunc
	sonar     *sonar.Sonar
	mu        sync.RWMutex
	members   map[string]coremodel.ClusterMember
}

// New will create a new MemberList instance using the given configuration.
func New(kv kv.KV, localPort int, sonar *sonar.Sonar) *MemberList {
	ctx, cancelFn := context.WithCancel(context.Background())
	return &MemberList{
		localPort: localPort,
		kv:        kv,
		members:   make(map[string]coremodel.ClusterMember),
		ctx:       ctx,
		ctxCancel: cancelFn,
		sonar:     sonar,
	}
}

// Start is used to join a cluster by registering instance member into the shared KV storage.
func (ml *MemberList) Start(ctx context.Context) error {
	if err := ml.join(ctx); err != nil {
		return err
	}
	// fetch current member list
	if err := ml.refreshMemberList(ctx); err != nil {
		return err
	}
	log.Infow("Registered local instance", "port", ml.localPort)

	return nil
}

// Stop unregisters instance member info from the cluster.
func (ml *MemberList) Stop(ctx context.Context) error {
	// stop watching changes...
	ml.ctxCancel()

	// unregister local instance
	if err := ml.kv.Del(ctx, localMemberKey()); err != nil {
		return err
	}
	log.Infow("Unregistered local instance", "port", ml.localPort)

	return nil
}

// GetMember returns cluster member info associated to an identifier.
func (ml *MemberList) GetMember(instanceID string) (m coremodel.ClusterMember, ok bool) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	m, ok = ml.members[instanceID]
	return
}

// GetMembers returns all cluster registered members.
func (ml *MemberList) GetMembers() map[string]coremodel.ClusterMember {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	res := make(map[string]coremodel.ClusterMember)
	for k, v := range ml.members {
		res[k] = v
	}
	return res
}

func (ml *MemberList) join(ctx context.Context) error {
	lm, err := ml.getLocalMember()
	if err != nil {
		return err
	}
	kvVal := fmt.Sprintf(memberValueFormat, lm.String(), lm.APIVer)
	return ml.kv.Put(ctx, localMemberKey(), kvVal)
}

func (ml *MemberList) refreshMemberList(ctx context.Context) error {
	ch := make(chan error, 1)

	go func() {
		wCh := ml.kv.Watch(ml.ctx, memberKeyPrefix, false)

		ms, err := ml.getMembers(ctx)
		if err != nil {
			ch <- err
			return
		}
		ml.mu.Lock()
		for _, m := range ms {
			ml.members[m.InstanceID] = m
		}
		ml.mu.Unlock()

		// post updated member list event
		err = ml.postUpdateEvent(ctx, &event.MemberListEventInfo{
			Registered: ms,
		})
		if err != nil {
			ch <- err
			return
		}
		close(ch) // signal update

		// watch changes
		for wResp := range wCh {
			if err := wResp.Err; err != nil {
				log.Warnf("Error occurred watching memberlist: %v", err)
				continue
			}
			// process changes
			if err := ml.processKVEvents(ctx, wResp.Events); err != nil {
				log.Warnf("Failed to process memberlist changes: %v", err)
			}
		}
	}()
	return <-ch
}

func (ml *MemberList) getMembers(ctx context.Context) ([]coremodel.ClusterMember, error) {
	vs, err := ml.kv.GetPrefix(ctx, memberKeyPrefix)
	if err != nil {
		return nil, err
	}
	res := make([]coremodel.ClusterMember, 0, len(vs))
	for k, val := range vs {
		if isLocalMemberKey(k) {
			continue // ignore local instance events
		}
		m, err := decodeClusterMember(k, string(val))
		if err != nil {
			log.Warnf("Failed to decode cluster member: %v", err)
			continue
		}
		if m == nil {
			continue // discard local instance
		}
		res = append(res, *m)
	}
	return res, nil
}

func (ml *MemberList) getLocalMember() (*coremodel.ClusterMember, error) {
	localIP, err := getLocalIP()
	if err != nil {
		return nil, err
	}
	return &coremodel.ClusterMember{
		InstanceID: instance.ID(),
		Host:       localIP,
		Port:       ml.localPort,
		APIVer:     version.ClusterAPIVersion,
	}, nil
}

func (ml *MemberList) processKVEvents(ctx context.Context, kvEvents []kv.WatchEvent) error {
	var putMembers []coremodel.ClusterMember
	var delMemberKeys []string

	ml.mu.Lock()
	for _, ev := range kvEvents {
		if isLocalMemberKey(ev.Key) {
			continue // ignore local instance events
		}
		switch ev.Type {
		case kv.Put:
			m, err := decodeClusterMember(ev.Key, string(ev.Val))
			if err != nil {
				return err
			}
			ml.members[m.InstanceID] = *m
			putMembers = append(putMembers, *m)

			log.Infow("Registered cluster member", "instance_id", m.InstanceID, "address", m.String(), "cluster_api_ver", m.APIVer.String())

		case kv.Del:
			memberKey := strings.TrimPrefix(ev.Key, memberKeyPrefix)
			delete(ml.members, memberKey)
			delMemberKeys = append(delMemberKeys, memberKey)

			log.Infow("Unregistered cluster member", "instance_id", memberKey)
		}
	}
	ml.mu.Unlock()

	// post updated event
	return ml.postUpdateEvent(ctx, &event.MemberListEventInfo{
		Registered:       putMembers,
		UnregisteredKeys: delMemberKeys,
	})
}

func (ml *MemberList) postUpdateEvent(ctx context.Context, evInf *event.MemberListEventInfo) error {
	e := sonar.NewEventBuilder(event.MemberListUpdated).
		WithInfo(evInf).
		WithSender(ml).
		Build()
	return ml.sonar.Post(ctx, e)
}

func decodeClusterMember(key, val string) (*coremodel.ClusterMember, error) {
	instanceID := strings.TrimPrefix(key, memberKeyPrefix)

	var addr, minClusterVer string
	_, _ = fmt.Sscanf(val, memberValueFormat, &addr, &minClusterVer)

	var major, minor, patch uint
	_, _ = fmt.Sscanf(minClusterVer, "v%d.%d.%d", &major, &minor, &patch)

	host, sPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(sPort)
	return &coremodel.ClusterMember{
		InstanceID: instanceID,
		Host:       host,
		Port:       port,
		APIVer:     version.NewVersion(major, minor, patch),
	}, nil
}

func getLocalIP() (string, error) {
	addrs, err := interfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", errors.New("memberlist: failed to get local ip")
}

func localMemberKey() string {
	return memberKeyPrefix + instance.ID()
}

func isLocalMemberKey(k string) bool {
	return k == localMemberKey()
}
