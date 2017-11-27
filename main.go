/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package main

import (
	"flag"
	"fmt"
	"os"

	"path/filepath"
	"strconv"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/server"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/version"
)

func main() {
	var configFile string
	var showVersion bool
	var showUsage bool

	flag.BoolVar(&showUsage, "help", false, "show application usage")
	flag.BoolVar(&showVersion, "version", false, "show application version")
	flag.StringVar(&configFile, "config", "/etc/jackal/jackal.yaml", "configuration path file")
	flag.Parse()

	// print usage
	if showUsage {
		flag.Usage()
		os.Exit(-1)
	}

	// print version
	if showVersion {
		fmt.Printf("jackal version: %v", version.ApplicationVersion)
		os.Exit(-1)
	}

	// load configuration
	if err := config.Load(configFile); err != nil {
		fmt.Fprintf(os.Stderr, "jackal: %v", err)
		os.Exit(-1)
	}
	if len(config.DefaultConfig.Servers) == 0 {
		fmt.Fprint(os.Stderr, "jackal: couldn't find a server configuration")
		os.Exit(-1)
	}

	// initialize logger subsystem
	if err := log.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "jackal: %v", err)
		os.Exit(-1)
	}

	// initialize storage subsystem
	storage.Instance()

	// create PID file
	if len(config.DefaultConfig.PIDFile) > 0 {
		if err := createPIDFile(config.DefaultConfig.PIDFile); err != nil {
			log.Warnf("%v", err)
		}
	}

	// start serving...
	log.Infof("jackal %v", version.ApplicationVersion)
	server.Initialize()
}

func createPIDFile(PIDFile string) error {
	if err := os.MkdirAll(filepath.Dir(PIDFile), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(PIDFile)
	if err != nil {
		return err
	}
	currentPid := os.Getpid()
	if _, err := file.WriteString(strconv.FormatInt(int64(currentPid), 10)); err != nil {
		return err
	}
	return nil
}
