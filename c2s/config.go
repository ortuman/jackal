/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"crypto/tls"

	"github.com/ortuman/jackal/auth"
	"github.com/ortuman/jackal/module"
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
	"github.com/ortuman/jackal/server/transport"
)

type Modules struct {
	Roster       *roster.Roster
	Offline      *offline.Offline
	LastActivity *xep0012.LastActivity
	DiscoInfo    *xep0030.DiscoInfo
	Private      *xep0049.Private
	VCard        *xep0054.VCard
	Register     *xep0077.Register
	Version      *xep0092.Version
	BlockingCmd  *xep0191.BlockingCommand
	Ping         *xep0199.Ping

	IQHandlers []module.IQHandler
}

type Config struct {
	TLSConfig      *tls.Config
	Transport      transport.Transport
	ConnectTimeout int
	MaxStanzaSize  int
	Authenticators []auth.Authenticator
	Modules        Modules
}
