/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof" // http profile handlers
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/version"
	"github.com/pkg/errors"
)

const (
	defaultShutDownWaitTime = time.Duration(5) * time.Second
)

var logoStr = []string{
	`        __               __            __   `,
	`       |__|____    ____ |  | _______  |  |  `,
	`       |  \__  \ _/ ___\|  |/ /\__  \ |  |  `,
	`       |  |/ __ \\  \___|    <  / __ \|  |__`,
	`   /\__|  (____  /\___  >__|_ \(____  /____/`,
	`   \______|    \/     \/     \/     \/      `,
}

const usageStr = `
Usage: jackal [options]

Server Options:
    -c, --Config <file>    Configuration file path
Common Options:
    -h, --help             Show this message
    -v, --version          Show version
`

// Application encapsulates a jackal server application.
type Application struct {
	output           io.Writer
	args             []string
	logger           log.Logger
	router           router.GlobalRouter
	mods             *module.Modules
	comps            *component.Components
	c2s              *c2s.C2S
	debugSrv         *http.Server
	waitStopCh       chan os.Signal
	shutDownWaitSecs time.Duration
}

// New returns a runnable application given an output and a command line arguments array.
func New(output io.Writer, args []string) *Application {
	return &Application{
		output:           output,
		args:             args,
		waitStopCh:       make(chan os.Signal, 1),
		shutDownWaitSecs: defaultShutDownWaitTime}
}

// Run runs jackal application until either a stop signal is received or an error occurs.
func (a *Application) Run() error {
	if len(a.args) == 0 {
		return errors.New("empty command-line arguments")
	}
	var configFile string
	var showVersion, showUsage bool

	fs := flag.NewFlagSet("jackal", flag.ExitOnError)
	fs.SetOutput(a.output)

	fs.BoolVar(&showUsage, "help", false, "Show this message")
	fs.BoolVar(&showUsage, "h", false, "Show this message")
	fs.BoolVar(&showVersion, "version", false, "Print version information.")
	fs.BoolVar(&showVersion, "v", false, "Print version information.")
	fs.StringVar(&configFile, "config", "/etc/jackal/jackal.yml", "Configuration file path.")
	fs.StringVar(&configFile, "c", "/etc/jackal/jackal.yml", "Configuration file path.")
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
		_, _ = fmt.Fprintf(a.output, "jackal version: %v\n", version.ApplicationVersion)
		return nil
	}
	// load configuration
	var cfg Config
	err := cfg.FromFile(configFile)
	if err != nil {
		return err
	}
	// create PID file
	if err := a.createPIDFile(cfg.PIDFile); err != nil {
		return err
	}
	// initialize logger
	err = a.initLogger(&cfg.Logger, a.output)
	if err != nil {
		return err
	}

	// show jackal's fancy logo
	a.printLogo()

	// initialize storage
	repContainer, err := storage.New(&cfg.Storage)
	if err != nil {
		return err
	}
	// initialize router
	a.router, err = router.New(&cfg.Router, repContainer.User(), repContainer.BlockList())
	if err != nil {
		return err
	}

	// initialize modules & components...
	a.mods = module.New(&cfg.Modules, a.router, repContainer)
	a.comps = component.New(&cfg.Components, a.mods.DiscoInfo)

	// start serving c2s...
	a.c2s, err = c2s.New(cfg.C2S, a.mods, a.comps, a.router, repContainer.User(), repContainer.BlockList())
	if err != nil {
		return err
	}
	a.c2s.Start()

	// initialize debug server...
	if cfg.Debug.Port > 0 {
		if err := a.initDebugServer(cfg.Debug.Port); err != nil {
			return err
		}
	}

	// ...wait for stop signal to shutdown
	sig := a.waitForStopSignal()
	log.Infof("received %s signal... shutting down...", sig.String())

	return a.gracefullyShutdown()
}

func (a *Application) showVersion() {
	_, _ = fmt.Fprintf(a.output, "jackal version: %v\n", version.ApplicationVersion)
}

func (a *Application) createPIDFile(pidFile string) error {
	if len(pidFile) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(pidFile), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(pidFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	currentPid := os.Getpid()
	if _, err := file.WriteString(strconv.FormatInt(int64(currentPid), 10)); err != nil {
		return err
	}
	return nil
}

func (a *Application) initLogger(config *loggerConfig, output io.Writer) error {
	var logFiles []io.WriteCloser
	if len(config.LogPath) > 0 {
		// create logFile intermediate directories.
		if err := os.MkdirAll(filepath.Dir(config.LogPath), os.ModePerm); err != nil {
			return err
		}
		f, err := os.OpenFile(config.LogPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		logFiles = append(logFiles, f)
	}
	l, err := log.New(config.Level, output, logFiles...)
	if err != nil {
		return err
	}
	a.logger = l
	log.Set(a.logger)
	return nil
}

func (a *Application) printLogo() {
	for i := range logoStr {
		log.Infof("%s", logoStr[i])
	}
	log.Infof("")
	log.Infof("jackal %v\n", version.ApplicationVersion)
}

func (a *Application) initDebugServer(port int) error {
	a.debugSrv = &http.Server{}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	go func() { _ = a.debugSrv.Serve(ln) }()
	log.Infof("debug server listening at %d...", port)
	return nil
}

func (a *Application) waitForStopSignal() os.Signal {
	signal.Notify(a.waitStopCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	return <-a.waitStopCh
}

func (a *Application) gracefullyShutdown() error {
	// wait until application has been shut down
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(a.shutDownWaitSecs))
	defer cancel()

	select {
	case <-a.shutdown(ctx):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Application) shutdown(ctx context.Context) <-chan bool {
	c := make(chan bool, 1)
	go func() {
		if a.debugSrv != nil {
			_ = a.debugSrv.Shutdown(ctx)
		}
		a.c2s.Shutdown(ctx)

		_ = a.comps.Shutdown(ctx)
		_ = a.mods.Shutdown(ctx)

		log.Unset()
		c <- true
	}()
	return c
}
