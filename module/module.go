/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"sync"

	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0012"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/module/xep0049"
	"github.com/ortuman/jackal/module/xep0054"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0191"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

// Module represents a generic XMPP module.
type Module interface {
}

// IQHandler represents an IQ handler module.
type IQHandler interface {
	Module

	// MatchesIQ returns whether or not an IQ should be
	// processed by the module.
	MatchesIQ(iq *xmpp.IQ) bool

	// ProcessIQ processes a module IQ taking according actions
	// over the associated stream.
	ProcessIQ(iq *xmpp.IQ, stm stream.C2S)
}

// Mods structure keeps reference to all active modules.
type Mods struct {
	Roster       *roster.Roster
	Offline      *offline.Offline
	LastActivity *xep0012.LastActivity
	Private      *xep0049.Private
	DiscoInfo    *xep0030.DiscoInfo
	VCard        *xep0054.VCard
	Register     *xep0077.Register
	Version      *xep0092.Version
	BlockingCmd  *xep0191.BlockingCommand
	Ping         *xep0199.Ping

	iqHandlers []IQHandler
	all        []Module
}

var (
	instMu      sync.RWMutex
	mods        Mods
	shutdownCh  chan struct{}
	initialized bool
)

// Initialize initializes module component.
func Initialize(cfg *Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	initializeModules(cfg)
	initialized = true
}

// Shutdown shuts down module sub system stopping every active module.
func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return
	}
	close(shutdownCh)
	mods = Mods{}
	initialized = false
}

// Modules returns current active modules.
func Modules() Mods {
	return mods
}

// ProcessIQ process a module IQ returning 'service unavailable'
// in case it can't be properly handled.
func ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	for _, handler := range mods.iqHandlers {
		if !handler.MatchesIQ(iq) {
			continue
		}
		handler.ProcessIQ(iq, stm)
		return
	}

	// ...IQ not handled...
	if iq.IsGet() || iq.IsSet() {
		stm.SendElement(iq.ServiceUnavailableError())
	}
}

func initializeModules(cfg *Config) {
	shutdownCh = make(chan struct{})

	// XEP-0030: Service Discovery (https://xmpp.org/extensions/xep-0030.html)
	mods.DiscoInfo = xep0030.New(shutdownCh)
	mods.iqHandlers = append(mods.iqHandlers, mods.DiscoInfo)
	mods.all = append(mods.all, mods.DiscoInfo)

	// Roster (https://xmpp.org/rfcs/rfc3921.html#roster)
	if _, ok := cfg.Enabled["roster"]; ok {
		mods.Roster = roster.New(&cfg.Roster, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.Roster)
		mods.all = append(mods.all, mods.Roster)
	}

	// XEP-0012: Last Activity (https://xmpp.org/extensions/xep-0012.html)
	if _, ok := cfg.Enabled["last_activity"]; ok {
		mods.LastActivity = xep0012.New(mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.LastActivity)
		mods.all = append(mods.all, mods.LastActivity)
	}

	// XEP-0049: Private XML Storage (https://xmpp.org/extensions/xep-0049.html)
	if _, ok := cfg.Enabled["private"]; ok {
		mods.Private = xep0049.New(shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.Private)
		mods.all = append(mods.all, mods.Private)
	}

	// XEP-0054: vcard-temp (https://xmpp.org/extensions/xep-0054.html)
	if _, ok := cfg.Enabled["vcard"]; ok {
		mods.VCard = xep0054.New(mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.VCard)
		mods.all = append(mods.all, mods.VCard)
	}

	// XEP-0077: In-band registration (https://xmpp.org/extensions/xep-0077.html)
	if _, ok := cfg.Enabled["registration"]; ok {
		mods.Register = xep0077.New(&cfg.Registration, mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.Register)
		mods.all = append(mods.all, mods.Register)
	}

	// XEP-0092: Software Version (https://xmpp.org/extensions/xep-0092.html)
	if _, ok := cfg.Enabled["version"]; ok {
		mods.Version = xep0092.New(&cfg.Version, mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.Version)
		mods.all = append(mods.all, mods.Version)
	}

	// XEP-0160: Offline message storage (https://xmpp.org/extensions/xep-0160.html)
	if _, ok := cfg.Enabled["offline"]; ok {
		mods.Offline = offline.New(&cfg.Offline, mods.DiscoInfo, shutdownCh)
		mods.all = append(mods.all, mods.Offline)
	}

	// XEP-0191: Blocking Command (https://xmpp.org/extensions/xep-0191.html)
	if _, ok := cfg.Enabled["blocking_command"]; ok {
		mods.BlockingCmd = xep0191.New(mods.DiscoInfo, mods.Roster, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.BlockingCmd)
		mods.all = append(mods.all, mods.BlockingCmd)
	}

	// XEP-0199: XMPP Ping (https://xmpp.org/extensions/xep-0199.html)
	if _, ok := cfg.Enabled["ping"]; ok {
		mods.Ping = xep0199.New(&cfg.Ping, mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.Ping)
		mods.all = append(mods.all, mods.Ping)
	}
}
