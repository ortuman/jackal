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

package etcdkv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	kvtypes "github.com/ortuman/jackal/pkg/cluster/kv/types"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	leaseTTLInSeconds int64 = 10

	etcdKVType = "etcd"
)

// Config contains etcd configuration parameters.
type Config struct {
	Endpoints            []string      `fig:"endpoints"`
	Username             string        `fig:"username"`
	Password             string        `fig:"password"`
	DialTimeout          time.Duration `fig:"dial_timeout" default:"20s"`
	DialKeepAliveTime    time.Duration `fig:"dial_keep_alive_time" default:"30s"`
	DialKeepAliveTimeout time.Duration `fig:"dial_keep_alive_timeout" default:"10s"`
	KeepAliveTime        time.Duration `fig:"keep_alive" default:"10s"`
	Timeout              time.Duration `fig:"keep_alive" default:"20m"`
}

// KV represents an etcd key-value store implementation.
type KV struct {
	cfg      Config
	logger   kitlog.Logger
	cli      *etcdv3.Client
	leaseID  etcdv3.LeaseID
	ctx      context.Context
	cancelFn context.CancelFunc
	kaCh     <-chan *etcdv3.LeaseKeepAliveResponse
	done     int32
}

// New returns a new etcd key-value store instance.
func New(cfg Config, logger kitlog.Logger) *KV {
	ctx, cancel := context.WithCancel(context.Background())
	return &KV{
		cfg:      cfg,
		logger:   logger,
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
func (k *KV) Watch(ctx context.Context, prefix string, withPrevVal bool) <-chan kvtypes.WatchResp {
	wCh := make(chan kvtypes.WatchResp)

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
	if err := k.checkHealth(k.cfg.Endpoints); err != nil {
		return err
	}
	// perform dialing
	cli, err := dial(k.cfg)
	if err != nil {
		return err
	}
	k.cli = cli

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

	level.Info(k.logger).Log("msg", "started kv store", "type", etcdKVType)
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
	if err := k.cli.Close(); err != nil {
		return err
	}
	level.Info(k.logger).Log("msg", "stopped kv store", "type", etcdKVType)
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
				level.Warn(k.logger).Log("msg", "failed to perform lease keepalive", "err", err, "lease_id", k.leaseID)

				retries++
				if retries == maxKeepAliveRetries {
					level.Error(k.logger).Log("msg", "unable to refresh lease TTL: max retries reached", "max_retries", maxKeepAliveRetries, "lease_id", k.leaseID)

					// shutdown process to avoid split-brain scenario
					shutdown()
					return
				}
			}
		}
	}
}

func (k *KV) checkHealth(endpoints []string) error {
	type healthResponse struct {
		Health string `json:"health"`
	}

	var errHealthCheckFailedFn = func(err error) error {
		return fmt.Errorf("etcd health check failed: %v", err)
	}
	for _, endpoint := range endpoints {
		resp, err := http.Get(fmt.Sprintf("%s/health", endpoint))
		if err != nil {
			return errHealthCheckFailedFn(err)
		}
		var hResp healthResponse
		if err := json.NewDecoder(resp.Body).Decode(&hResp); err != nil {
			_ = resp.Body.Close()
			return errHealthCheckFailedFn(err)
		}
		_ = resp.Body.Close()

		healthy, _ := strconv.ParseBool(hResp.Health)
		if !healthy {
			return errHealthCheckFailedFn(fmt.Errorf("health = false, for endpoint %s", endpoint))
		}
	}
	return nil
}

func toWatchResp(wResp *etcdv3.WatchResponse) kvtypes.WatchResp {
	var events []kvtypes.WatchEvent
	for _, ev := range wResp.Events {
		var kvEvent kvtypes.WatchEvent

		switch ev.Type {
		case etcdv3.EventTypePut:
			kvEvent.Type = kvtypes.Put
		case etcdv3.EventTypeDelete:
			kvEvent.Type = kvtypes.Del
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
	return kvtypes.WatchResp{
		Events: events,
		Err:    wResp.Err(),
	}
}

func shutdown() {
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)
}

func dial(cfg Config) (*etcdv3.Client, error) {
	dialOptions := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.KeepAliveTime,
			Timeout:             cfg.Timeout,
			PermitWithoutStream: true,
		}),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	return etcdv3.New(etcdv3.Config{
		Endpoints:            cfg.Endpoints,
		Username:             cfg.Username,
		Password:             cfg.Password,
		DialTimeout:          cfg.DialTimeout,
		DialKeepAliveTime:    cfg.DialKeepAliveTime,
		DialKeepAliveTimeout: cfg.DialKeepAliveTimeout,
		DialOptions:          dialOptions,
	})
}
