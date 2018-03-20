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

package main

import (
	"context"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	clusterconnmanager "github.com/ortuman/jackal/cluster/connmanager"
	"github.com/ortuman/jackal/cluster/kv"
	etcdkv "github.com/ortuman/jackal/cluster/kv/etcd"
	etcdlocker "github.com/ortuman/jackal/cluster/locker/etcd"
	"github.com/ortuman/jackal/cluster/memberlist"
)

const etcdMemberListTimeout = time.Second * 5

func initEtcd(a *serverApp, cfg etcdConfig) error {
	cli, err := etcdv3.New(etcdv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), etcdMemberListTimeout)
	defer cancel()

	// obtain memberlist to check cluster health
	_, err = cli.MemberList(ctx)
	if err != nil {
		return err
	}
	a.etcdCli = cli
	return nil
}

func initLocker(a *serverApp) {
	a.locker = etcdlocker.New(a.etcdCli)
	a.registerStartStopper(a.locker)
}

func initKVStore(a *serverApp) {
	etcdKV := etcdkv.New(a.etcdCli)
	a.kv = kv.NewMeasured(etcdKV)
	a.registerStartStopper(a.kv)
}

func initMemberList(a *serverApp, clusterPort int) {
	a.memberList = memberlist.New(a.kv, clusterPort, a.sonar)
	a.registerStartStopper(a.memberList)
	return
}

func initClusterConnManager(a *serverApp) {
	a.clusterConnMng = clusterconnmanager.NewManager(a.sonar)
	a.registerStartStopper(a.clusterConnMng)
}
