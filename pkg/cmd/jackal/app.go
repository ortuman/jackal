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
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	adminserver "github.com/ortuman/jackal/pkg/admin/server"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/c2s"
	clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	etcdkv "github.com/ortuman/jackal/pkg/cluster/kv/etcd"
	"github.com/ortuman/jackal/pkg/cluster/locker"
	etcdlocker "github.com/ortuman/jackal/pkg/cluster/locker/etcd"
	"github.com/ortuman/jackal/pkg/cluster/memberlist"
	clusterrouter "github.com/ortuman/jackal/pkg/cluster/router"
	clusterserver "github.com/ortuman/jackal/pkg/cluster/server"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/component/extcomponentmanager"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/log/zap"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/repository"
	measuredrepository "github.com/ortuman/jackal/pkg/repository/measured"
	pgsqlrepository "github.com/ortuman/jackal/pkg/repository/pgsql"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/s2s"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/util/crashreporter"
	"github.com/ortuman/jackal/pkg/util/stringmatcher"
	tlsutil "github.com/ortuman/jackal/pkg/util/tls"
	"github.com/ortuman/jackal/pkg/version"
)

const (
	darwinOpenMax = 10240

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

	peppers *pepper.Keys
	hk      *hook.Hooks

	etcdCli *etcdv3.Client
	locker  locker.Locker
	kv      kv.KV

	rep        repository.Repository
	memberList *memberlist.MemberList
	resMng     *c2s.ResourceManager

	shapers        shaper.Shapers
	hosts          *host.Hosts
	clusterConnMng *clusterconnmanager.Manager
	localRouter    *c2s.LocalRouter
	clusterRouter  *clusterrouter.Router
	s2sOutProvider *s2s.OutProvider
	s2sInHub       *s2s.InHub
	router         router.Router
	mods           *module.Modules
	comps          *component.Components
	extCompMng     *extcomponentmanager.Manager

	starters []starter
	stoppers []stopper

	waitStopCh chan os.Signal
}

func run(output io.Writer, args []string) error {
	// Seed the math/rand RNG from crypto/rand.
	rand.Seed(time.Now().UnixNano())

	defer crashreporter.RecoverAndReportPanic()

	panic("foo panic")

	a := &serverApp{
		output:     output,
		args:       args,
		waitStopCh: make(chan os.Signal, 1),
	}
	fs := flag.NewFlagSet("jackal", flag.ExitOnError)
	fs.SetOutput(a.output)

	var configFile string
	var showVersion, showUsage bool

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
	// if present, override config file url with env var
	if envCfgFile := os.Getenv(envConfigFile); len(envCfgFile) > 0 {
		configFile = envCfgFile
	}
	// load configuration
	cfg, err := loadConfig(configFile)
	if err != nil {
		return err
	}
	// enable gRPC prometheus histograms
	grpc_prometheus.EnableHandlingTimeHistogram()

	// set maximum opened files limit
	if err := setRLimit(); err != nil {
		return err
	}
	// init logger
	log.SetLogger(
		zap.NewLogger(cfg.Logger.OutputPath),
		cfg.Logger.Level,
	)

	log.Infow("Jackal is starting...",
		"version", version.Version,
		"go_ver", runtime.Version(),
		"go_os", runtime.GOOS,
		"go_arch", runtime.GOARCH,
	)

	// init pepper keys
	peppers, err := pepper.NewKeys(cfg.Peppers.Keys, cfg.Peppers.UseID)
	if err != nil {
		return err
	}
	a.peppers = peppers

	// init hooks
	a.hk = hook.NewHooks()

	// init etcd
	if err := a.initEtcd(cfg.Cluster.Etcd); err != nil {
		return err
	}
	a.initLocker()
	a.initKVStore()

	// init cluster connection manager
	a.initClusterConnManager()

	// init repository
	if err := a.initRepository(cfg.Storage); err != nil {
		return err
	}
	// init C2S/S2S routers
	if err := a.initHosts(cfg.Hosts); err != nil {
		return err
	}
	if err := a.initShapers(cfg.Shapers); err != nil {
		return err
	}
	a.initS2S(cfg.S2SOut)
	a.initRouters()

	// init components & modules
	a.initComponents(cfg.Components)

	if err := a.initModules(cfg.Modules); err != nil {
		return err
	}
	// init HTTP server
	a.registerStartStopper(newHTTPServer(cfg.HTTPPort))

	// init admin server
	if !cfg.Admin.Disabled {
		a.initAdminServer(cfg.Admin.BindAddr, cfg.Admin.Port)
	}
	// init cluster server
	a.initClusterServer(cfg.Cluster.BindAddr, cfg.Cluster.Port)

	// init memberlist
	a.initMemberList(cfg.Cluster.Port)

	// init C2S/S2S listeners
	if err := a.initListeners(cfg.Listeners); err != nil {
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

func (a *serverApp) initEtcd(cfg etcdConfig) error {
	const etcdMemberListTimeout = time.Second * 5
	cli, err := etcdv3.New(etcdv3.Config{
		Endpoints:         cfg.Endpoints,
		DialTimeout:       cfg.DialTimeout,
		DialKeepAliveTime: time.Second,
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

func (a *serverApp) initLocker() {
	a.locker = etcdlocker.New(a.etcdCli)
	a.registerStartStopper(a.locker)
}

func (a *serverApp) initKVStore() {
	etcdKV := etcdkv.New(a.etcdCli)
	a.kv = kv.NewMeasured(etcdKV)
	a.registerStartStopper(a.kv)
}

func (a *serverApp) initClusterConnManager() {
	a.clusterConnMng = clusterconnmanager.NewManager(a.hk)
	a.registerStartStopper(a.clusterConnMng)
}

func (a *serverApp) initRepository(sCfg storageConfig) error {
	cfg := sCfg.PgSQL
	opts := pgsqlrepository.Config{
		MaxIdleConns:    cfg.MaxIdleConns,
		MaxOpenConns:    cfg.MaxOpenConns,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}
	pgRep := pgsqlrepository.New(
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
		opts,
	)
	a.rep = measuredrepository.New(pgRep)
	a.registerStartStopper(a.rep)
	return nil
}

func (a *serverApp) initHosts(configs []hostConfig) error {
	const defaultDomain = "localhost"
	h := host.New()
	if len(configs) == 0 {
		cer, err := tlsutil.LoadCertificate("", "", defaultDomain)
		if err != nil {
			return err
		}
		h.RegisterDefaultHost(defaultDomain, cer)
		a.hosts = h
		return nil
	}
	for i, config := range configs {
		cer, err := tlsutil.LoadCertificate(config.TLS.PrivateKeyFile, config.TLS.CertFile, config.Domain)
		if err != nil {
			return err
		}
		if i == 0 {
			h.RegisterDefaultHost(config.Domain, cer)
		} else {
			h.RegisterHost(config.Domain, cer)
		}
	}
	a.hosts = h
	return nil
}

func (a *serverApp) initShapers(cfgs []shaperConfig) error {
	a.shapers = make(shaper.Shapers, 0)
	for _, cfg := range cfgs {
		var sm stringmatcher.Matcher
		switch {
		case len(cfg.Matching.JID.In) > 0:
			sm = stringmatcher.NewStringMatcher(cfg.Matching.JID.In)
		case len(cfg.Matching.JID.RegEx) > 0:
			var err error
			sm, err = stringmatcher.NewRegExMatcher(cfg.Matching.JID.RegEx)
			if err != nil {
				return err
			}
		default:
			sm = stringmatcher.Any
		}
		a.shapers = append(a.shapers, shaper.New(cfg.MaxSessions, cfg.Rate.Limit, cfg.Rate.Burst, sm))

		log.Infow(fmt.Sprintf("Registered '%s' shaper configuration", cfg.Name),
			"name", cfg.Name,
			"max_sessions", cfg.MaxSessions,
			"limit", cfg.Rate.Limit,
			"burst", cfg.Rate.Burst)
	}
	return nil
}

func (a *serverApp) initMemberList(clusterPort int) {
	a.memberList = memberlist.New(a.kv, clusterPort, a.hk)
	a.registerStartStopper(a.memberList)
	return
}

func (a *serverApp) initListeners(configs []listenerConfig) error {
	for _, cfg := range configs {
		lnFn, ok := lnFns[cfg.Type]
		if !ok {
			return fmt.Errorf("main: unrecognized listener: %s", cfg.Type)
		}
		ln := lnFn(a, cfg)
		a.registerStartStopper(ln)
	}
	return nil
}

func (a *serverApp) initS2S(cfg s2sOutConfig) {
	a.s2sOutProvider = s2s.NewOutProvider(a.hosts, a.kv, a.shapers, a.hk, s2s.Config{
		DialTimeout:      cfg.DialTimeout,
		DialbackSecret:   cfg.DialbackSecret,
		ConnectTimeout:   cfg.ConnectTimeout,
		KeepAliveTimeout: cfg.KeepAliveTimeout,
		RequestTimeout:   cfg.RequestTimeout,
		MaxStanzaSize:    cfg.MaxStanzaSize,
	})
	a.s2sInHub = s2s.NewInHub()

	a.registerStartStopper(a.s2sOutProvider)
	a.registerStartStopper(a.s2sInHub)
}

func (a *serverApp) initRouters() {
	// init shared resource hub
	a.resMng = c2s.NewResourceManager(a.kv)

	// init C2S router
	a.localRouter = c2s.NewLocalRouter(a.hosts)
	a.clusterRouter = clusterrouter.New(a.clusterConnMng)

	c2sRouter := c2s.NewRouter(a.localRouter, a.clusterRouter, a.resMng, a.rep, a.hk)
	s2sRouter := s2s.NewRouter(a.s2sOutProvider)

	// init global router
	a.router = router.New(a.hosts, c2sRouter, s2sRouter)

	a.registerStartStopper(a.router)
	return
}

func (a *serverApp) initComponents(_ componentsConfig) {
	a.comps = component.NewComponents(nil, a.hk)
	a.extCompMng = extcomponentmanager.New(a.kv, a.clusterConnMng, a.comps)

	a.registerStartStopper(a.comps)
	a.registerStartStopper(a.extCompMng)
}

func (a *serverApp) initModules(cfg modulesConfig) error {
	var mods []module.Module

	// enabled modules
	enabled := cfg.Enabled
	if len(enabled) == 0 {
		enabled = defaultModules
	}
	for _, mName := range enabled {
		fn, ok := modFns[mName]
		if !ok {
			return fmt.Errorf("main: unrecognized module name: %s", mName)
		}
		mods = append(mods, fn(a, cfg))
	}
	a.mods = module.NewModules(mods, a.hosts, a.router, a.hk)
	a.registerStartStopper(a.mods)
	return nil
}

func (a *serverApp) initAdminServer(bindAddr string, port int) {
	adminSrv := adminserver.New(bindAddr, port, a.rep, a.peppers, a.hk)
	a.registerStartStopper(adminSrv)
}

func (a *serverApp) initClusterServer(bindAddr string, port int) {
	clusterSrv := clusterserver.New(bindAddr, port, a.localRouter, a.comps)
	a.registerStartStopper(clusterSrv)
	return
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
