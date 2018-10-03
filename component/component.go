/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package component

import (
	"fmt"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

// Component represents a generic component interface.
type Component interface {
	Host() string
	ProcessStanza(stanza xmpp.Stanza, stm stream.C2S)
}

// singleton interface
var (
	instMu      sync.RWMutex
	comps       map[string]Component
	shutdownCh  chan struct{}
	initialized bool
)

// Initialize initializes the components manager.
func Initialize(cfg *Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	shutdownCh = make(chan struct{})

	cs := loadComponents(cfg)

	comps = make(map[string]Component)
	for _, c := range cs {
		host := c.Host()
		if _, ok := comps[host]; ok {
			log.Fatalf("%v", fmt.Errorf("component host name conflict: %s", host))
		}
		comps[host] = c
	}
	initialized = true
}

// Shutdown shuts down components manager system.
// This method should be used only for testing purposes.
func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return
	}
	close(shutdownCh)
	comps = nil
	initialized = false
}

// Get returns a specific component associated to host name.
func Get(host string) Component {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return nil
	}
	return comps[host]
}

// GetAll returns all initialized components.
func GetAll() []Component {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return nil
	}
	var ret []Component
	for _, comp := range comps {
		ret = append(ret, comp)
	}
	return ret
}

func loadComponents(cfg *Config) []Component {
	var ret []Component
	/*
		discoInfo := module.Modules().DiscoInfo
		if cfg.HttpUpload != nil {
			ret = append(ret, httpupload.New(cfg.HttpUpload, discoInfo, shutdownCh))
		}
	*/
	return ret
}
