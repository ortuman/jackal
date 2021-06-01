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

package etcdkv

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/log"
)

const (
	leaseTTLInSeconds int64 = 15
)

// KV represents an etcd key-value store implementation.
type KV struct {
	cli      *etcdv3.Client
	leaseID  etcdv3.LeaseID
	ctx      context.Context
	cancelFn context.CancelFunc
	kaCh     <-chan *etcdv3.LeaseKeepAliveResponse
	done     int32
}

// New returns a new etcd key-value store instance.
func New(cli *etcdv3.Client) *KV {
	ctx, cancel := context.WithCancel(context.Background())
	return &KV{
		cli:      cli,
		ctx:      ctx,
		cancelFn: cancel,
	}
}

// Put stores a new value associated to a given key.
func (k *KV) Put(ctx context.Context, key string, value string) error {
	_, err := k.cli.Put(ctx, key, value, etcdv3.WithLease(k.leaseID))
	return err
}

// Get retrieves a value associated to a given key.
func (k *KV) Get(ctx context.Context, key string) ([]byte, error) {
	getResp, err := k.cli.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(getResp.Kvs) == 0 {
		return nil, nil
	}
	return getResp.Kvs[0].Value, nil
}

// GetPrefix retrieves all values whose key matches prefix.
func (k *KV) GetPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	getResp, err := k.cli.Get(ctx, prefix, etcdv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	values := make(map[string][]byte, len(getResp.Kvs))
	for _, kVal := range getResp.Kvs {
		values[string(kVal.Key)] = kVal.Value
	}
	return values, nil
}

// Del deletes a value associated to a given key.
func (k *KV) Del(ctx context.Context, key string) error {
	_, err := k.cli.Delete(ctx, key)
	if err != nil {
		return err
	}
	return nil
}

// Watch watches on a key or prefix.
func (k *KV) Watch(ctx context.Context, prefix string, withPrevVal bool) <-chan kv.WatchResp {
	wCh := make(chan kv.WatchResp)

	var opts = []etcdv3.OpOption{
		etcdv3.WithPrefix(),
	}
	if withPrevVal {
		opts = append(opts, etcdv3.WithPrevKV())
	}
	watchResp := k.cli.Watch(ctx, prefix, opts...)
	go func() {
		for resp := range watchResp {
			wCh <- toWatchResp(&resp)
		}
		close(wCh)
	}()
	return wCh
}

// Start initializes etcd key-value store.
func (k *KV) Start(ctx context.Context) error {
	// create shared KV lease
	resp, err := k.cli.Grant(ctx, leaseTTLInSeconds)
	if err != nil {
		return err
	}
	k.leaseID = resp.ID

	k.kaCh, err = k.cli.KeepAlive(k.ctx, k.leaseID)
	if err != nil {
		return err
	}
	go k.keepAliveLease()

	log.Infow("Started etcd KV store")
	return nil
}

// Stop closes etcd underlying connection.
func (k *KV) Stop(ctx context.Context) error {
	atomic.StoreInt32(&k.done, 1)

	// stop refreshing lease TTL
	k.cancelFn()

	_, err := k.cli.Revoke(ctx, k.leaseID)
	if err != nil {
		return err
	}
	log.Infow("Stopped etcd KV store")
	return nil
}

func (k *KV) keepAliveLease() {
	const maxKeepAliveRetries = 10

	var err error
	var retries int
	for resp := range k.kaCh {
		if atomic.LoadInt32(&k.done) == 1 {
			return
		}
		if resp == nil {
			k.kaCh, err = k.cli.KeepAlive(k.ctx, k.leaseID)
			switch err {
			case nil:
				retries = 0

			default:
				log.Warnw(fmt.Sprintf("Failed to perform lease keepalive: %v", err), "lease_id", k.leaseID)

				retries++
				if retries == maxKeepAliveRetries {
					log.Errorf(fmt.Sprintf("Unable to refresh lease keepalive: max retries reached: %d", maxKeepAliveRetries), "lease_id", k.leaseID)

					// shutdown process to avoid split-brain scenario
					shutdown()
					return
				}
			}
		}
	}
}

func toWatchResp(wResp *etcdv3.WatchResponse) kv.WatchResp {
	var events []kv.WatchEvent
	for _, ev := range wResp.Events {
		var kvEvent kv.WatchEvent

		switch ev.Type {
		case etcdv3.EventTypePut:
			kvEvent.Type = kv.Put
		case etcdv3.EventTypeDelete:
			kvEvent.Type = kv.Del
		default:
			continue // unrecognized event type
		}
		kvEvent.Key = string(ev.Kv.Key)
		if len(ev.Kv.Value) > 0 {
			kvEvent.Val = ev.Kv.Value
		}
		if ev.PrevKv != nil && len(ev.PrevKv.Value) > 0 {
			kvEvent.PrevVal = ev.PrevKv.Value
		}
		events = append(events, kvEvent)
	}
	return kv.WatchResp{
		Events: events,
		Err:    wResp.Err(),
	}
}

func shutdown() {
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)
}
