/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/version"
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
    -c, --config <file>    Configuration file path
Common Options:
    -h, --help             Show this message
    -v, --version          Show version
`

func main() {
	var configFile string
	var showVersion bool
	var showUsage bool

	flag.BoolVar(&showUsage, "help", false, "Show this message")
	flag.BoolVar(&showUsage, "h", false, "Show this message")
	flag.BoolVar(&showVersion, "version", false, "Print version information.")
	flag.BoolVar(&showVersion, "v", false, "Print version information.")
	flag.StringVar(&configFile, "config", "/etc/jackal/jackal.yml", "Configuration file path.")
	flag.StringVar(&configFile, "c", "/etc/jackal/jackal.yml", "Configuration file path.")
	flag.Usage = func() {
		for i := range logoStr {
			fmt.Fprintf(os.Stdout, "%s\n", logoStr[i])
		}
		fmt.Fprintf(os.Stdout, "%s\n", usageStr)
	}
	flag.Parse()

	// print usage
	if showUsage {
		flag.Usage()
		return
	}

	// print version
	if showVersion {
		fmt.Fprintf(os.Stdout, "jackal version: %v\n", version.ApplicationVersion)
		return
	}
	// load configuration
	var cfg Config
	if err := cfg.FromFile(configFile); err != nil {
		fmt.Fprintf(os.Stderr, "jackal: %v\n", err)
		return
	}
	// initialize subsystems... (order matters)
	log.Initialize(&cfg.Logger)

	storage.Initialize(&cfg.Storage)

	host.Initialize(cfg.Hosts)

	router.Initialize(&router.Config{GetS2SOut: s2s.GetS2SOut})

	// initialize modules & components...
	module.Initialize(&cfg.Modules)
	component.Initialize(&cfg.Components)

	// create PID file
	if err := createPIDFile(cfg.PIDFile); err != nil {
		log.Warnf("%v", err)
	}
	// start serving...
	for i := range logoStr {
		log.Infof("%s", logoStr[i])
	}
	log.Infof("")
	log.Infof("jackal %v\n", version.ApplicationVersion)

	// initialize debug server...
	if cfg.Debug.Port > 0 {
		go initDebugServer(cfg.Debug.Port)
	}

	// start serving s2s...
	s2s.Initialize(cfg.S2S)

	// start serving c2s...
	c2s.Initialize(cfg.C2S)
}

var debugSrv *http.Server

func initDebugServer(port int) {
	debugSrv = &http.Server{}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("%v", err)
	}
	debugSrv.Serve(ln)
}

func createPIDFile(pidFile string) error {
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
	currentPid := os.Getpid()
	if _, err := file.WriteString(strconv.FormatInt(int64(currentPid), 10)); err != nil {
		return err
	}
	return nil
}
