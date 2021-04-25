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
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jackal-xmpp/sonar"
	adminserver "github.com/ortuman/jackal/admin/server"
	"github.com/ortuman/jackal/auth/pepper"
	"github.com/ortuman/jackal/c2s"
	clusterconnmanager "github.com/ortuman/jackal/cluster/connmanager"
	"github.com/ortuman/jackal/cluster/kv"
	"github.com/ortuman/jackal/cluster/locker"
	"github.com/ortuman/jackal/cluster/memberlist"
	clusterrouter "github.com/ortuman/jackal/cluster/router"
	clusterserver "github.com/ortuman/jackal/cluster/server"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/component/extcomponentmanager"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/log/zap"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/shaper"
	"github.com/ortuman/jackal/version"
)

const (
	darwinOpenMax = 10240

	defaultDomain = "localhost"

	defaultBootstrapTimeout = time.Minute
	defaultShutdownTimeout  = time.Second * 30

	envConfigFile = "JACKAL_CONFIG_FILE"
)

var logoStr = []string{
	`     __               __            __   `,
	`    |__|____    ____ |  | _______  |  |  `,
	`    |  \__  \ _/ ___\|  |/ /\__  \ |  |  `,
	`    |  |/ __ \\  \___|    <  / __ \|  |__`,
	`/\__|  (____  /\___  >__|_ \(____  /____/`,
	`\______|    \/     \/     \/     \/      `,
}

const usageStr = `
Usage: jackal [options]
Server Options:
    --config <file>    Configuration file path
Common Options:
    --help             Show this message
`

type starter interface {
	Start(ctx context.Context) error
}

type stopper interface {
	Stop(ctx context.Context) error
}

type startStopper interface {
	starter
	stopper
}

type serverApp struct {
	output io.Writer
	args   []string

	etcdCli *etcdv3.Client
	locker  locker.Locker
	kv      kv.KV

	peppers       *pepper.Keys
	sonar         *sonar.Sonar
	rep           repository.Repository
	adminServer   *adminserver.Server
	memberList    *memberlist.MemberList
	resMng        *c2s.ResourceManager
	clusterServer *clusterserver.Server

	shapers        shaper.Shapers
	hosts          *host.Hosts
	clusterConnMng *clusterconnmanager.Manager
	localRouter    *c2s.LocalRouter
	clusterRouter  *clusterrouter.Router
	s2sOutProvider *s2s.OutProvider
	s2sInHub       *s2s.InHub
	c2sRouter      router.C2SRouter
	s2sRouter      router.S2SRouter
	router         router.Router
	mods           *module.Modules
	comps          *component.Components
	extCompMng     *extcomponentmanager.Manager

	starters []starter
	stoppers []stopper

	waitStopCh chan os.Signal
}

func run(output io.Writer, args []string) error {
	var configFile string
	var showVersion, showUsage bool

	a := &serverApp{
		output:     output,
		args:       args,
		waitStopCh: make(chan os.Signal, 1),
	}
	fs := flag.NewFlagSet("jackal", flag.ExitOnError)
	fs.SetOutput(a.output)

	fs.BoolVar(&showUsage, "help", false, "Show this message")
	fs.BoolVar(&showVersion, "version", false, "Print version information.")
	fs.StringVar(&configFile, "config", "config.yaml", "Configuration file path.")
	fs.Usage = func() {
		for i := range logoStr {
			_, _ = fmt.Fprintf(a.output, "%s\n", logoStr[i])
		}
		_, _ = fmt.Fprintf(a.output, "%s\n", usageStr)
	}
	_ = fs.Parse(a.args[1:])

	// print usage
	if showUsage {
		fs.Usage()
		return nil
	}
	// print version
	if showVersion {
		_, _ = fmt.Fprintf(a.output, "jackal version: %v\n", version.Version)
		return nil
	}
	// set maximum opened files limit
	if err := setRLimit(); err != nil {
		return err
	}
	// enable gRPC prometheus histograms
	grpc_prometheus.EnableHandlingTimeHistogram()

	// if present, override config file url with env var
	if envCfgFile := os.Getenv(envConfigFile); len(envCfgFile) > 0 {
		configFile = envCfgFile
	}
	// load configuration
	cfg, err := loadConfig(configFile)
	if err != nil {
		return err
	}
	// init logger
	log.SetLogger(
		zap.NewLogger(cfg.Logger.OutputPath),
		cfg.Logger.Level,
	)
	// init etcd
	if err := initEtcd(a, cfg.Cluster.Etcd); err != nil {
		return err
	}
	initLocker(a)
	initKVStore(a)

	// init pepper keys
	peppers, err := pepper.NewKeys(cfg.Peppers.Keys, cfg.Peppers.UseID)
	if err != nil {
		return err
	}
	a.peppers = peppers

	// init sonar hub
	a.sonar = sonar.New()

	log.Infow("Jackal is starting...",
		"version", version.Version,
		"go_ver", runtime.Version(),
		"go_os", runtime.GOOS,
		"go_arch", runtime.GOARCH,
	)
	// init HTTP server
	a.registerStartStopper(newHTTPServer(cfg.HTTPPort))

	// init repository
	if err := initRepository(a, cfg.Storage); err != nil {
		return err
	}
	// init admin server
	initAdminServer(a, cfg.Admin)

	// init cluster connection manager
	initClusterConnManager(a)

	// init C2S/S2S routers
	if err := initHosts(a, cfg.Hosts); err != nil {
		return err
	}
	if err := initShapers(a, cfg.Shapers); err != nil {
		return err
	}
	initS2S(a, cfg.S2SOut)
	initRouters(a)

	// init components & modules
	initComponents(a, cfg.Components)

	if err := initModules(a, cfg.Modules); err != nil {
		return err
	}
	// init cluster server
	initClusterServer(a, cfg.Cluster.BindAddr, cfg.Cluster.Port)

	// init memberlist
	initMemberList(a, cfg.Cluster.Port)

	// init C2S/S2S listeners
	if err := initListeners(a, cfg.Listeners); err != nil {
		return err
	}

	if err := a.bootstrap(); err != nil {
		return err
	}
	// ...wait for stop signal to shutdown
	sig := a.waitForStopSignal()
	log.Infof("Received %s signal... shutting down...", sig.String())

	return a.shutdown()
}

func (a *serverApp) registerStartStopper(ss startStopper) {
	a.starters = append(a.starters, ss)
	a.stoppers = append([]stopper{ss}, a.stoppers...)
}

func (a *serverApp) bootstrap() error {
	// spin up all service subsystems
	ctx, cancel := context.WithTimeout(context.Background(), defaultBootstrapTimeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		// invoke all registered starters...
		for _, s := range a.starters {
			if err := s.Start(ctx); err != nil {
				errCh <- err
				return
			}
		}
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *serverApp) shutdown() error {
	// wait until shutdown has been completed
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		// invoke all registered stoppers...
		for _, st := range a.stoppers {
			if err := st.Stop(ctx); err != nil {
				errCh <- err
				return
			}
		}
		_ = a.etcdCli.Close()
		log.Close()
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *serverApp) waitForStopSignal() os.Signal {
	signal.Notify(a.waitStopCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	return <-a.waitStopCh
}

func setRLimit() error {
	var rLim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLim); err != nil {
		return err
	}
	if rLim.Cur < rLim.Max {
		switch runtime.GOOS {
		case "darwin":
			// The max file limit is 10240, even though
			// the max returned by Getrlimit is 1<<63-1.
			// This is OPEN_MAX in sys/syslimits.h.
			rLim.Cur = darwinOpenMax
		default:
			rLim.Cur = rLim.Max
		}
		return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLim)
	}
	return nil
}
