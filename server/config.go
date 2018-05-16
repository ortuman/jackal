/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/server/transport"
)

const (
	defaultTransportPort           = 5222
	defaultTransportMaxStanzaSize  = 32768
	defaultTransportConnectTimeout = 5
	defaultTransportKeepAlive      = 120
)

// ServerType represents a server type (c2s, s2s).
type ServerType int

const (
	// C2SServerType represents a client to client server type.
	C2SServerType ServerType = iota
	// S2SServerType represents a server-to-client server type.
	S2SServerType
)

// String returns ServerType string representation.
func (st ServerType) String() string {
	switch st {
	case C2SServerType:
		return "c2s"
	case S2SServerType:
		return "s2s"
	}
	return ""
}

// ResourceConflictPolicy represents a resource conflict policy.
type ResourceConflictPolicy int

const (
	// Override represents 'override' resource conflict policy.
	Override ResourceConflictPolicy = iota

	// Reject represents 'reject' resource conflict policy.
	Reject

	// Replace represents 'replace' resource conflict policy.
	Replace
)

// Config represents an XMPP server configuration.
type Config struct {
	ID               string
	Type             ServerType
	ResourceConflict ResourceConflictPolicy
	Transport        TransportConfig
	SASL             []string
	TLS              TLSConfig
	Modules          map[string]struct{}
	Compression      CompressConfig
	ModRoster        roster.Config
	ModOffline       offline.Config
	ModRegistration  xep0077.Config
	ModVersion       xep0092.Config
	ModPing          xep0199.Config
}

type configProxyType struct {
	ID               string          `yaml:"id"`
	Type             string          `yaml:"type"`
	ResourceConflict string          `yaml:"resource_conflict"`
	Transport        TransportConfig `yaml:"transport"`
	SASL             []string        `yaml:"sasl"`
	TLS              TLSConfig       `yaml:"tls"`
	Modules          []string        `yaml:"modules"`
	Compression      CompressConfig  `yaml:"compression"`
	ModRoster        roster.Config   `yaml:"mod_roster"`
	ModOffline       offline.Config  `yaml:"mod_offline"`
	ModRegistration  xep0077.Config  `yaml:"mod_registration"`
	ModVersion       xep0092.Config  `yaml:"mod_version"`
	ModPing          xep0199.Config  `yaml:"mod_ping"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (cfg *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	// validate server type
	switch strings.ToLower(p.Type) {
	case "c2s":
		cfg.Type = C2SServerType
	case "s2s":
		return errors.New("server.Config: s2s server type not yet supported")
	default:
		return fmt.Errorf("server.Config: unrecognized server type: %s", p.Type)
	}
	// validate resource conflict policy type
	rc := strings.ToLower(p.ResourceConflict)
	switch rc {
	case "override":
		cfg.ResourceConflict = Override
	case "reject":
		cfg.ResourceConflict = Reject
	case "", "replace":
		cfg.ResourceConflict = Replace
	default:
		return fmt.Errorf("invalid resource_conflict option: %s", rc)
	}
	// validate SASL mechanisms
	for _, sasl := range p.SASL {
		switch sasl {
		case "plain", "digest_md5", "scram_sha_1", "scram_sha_256":
			continue
		default:
			return fmt.Errorf("server.Config: unrecognized SASL mechanism: %s", sasl)
		}
	}
	// validate modules
	cfg.Modules = map[string]struct{}{}
	for _, module := range p.Modules {
		switch module {
		case "roster", "last_activity", "private", "vcard", "registration", "version", "blocking_command", "ping",
			"offline":
			break
		default:
			return fmt.Errorf("config.Server: unrecognized module: %s", module)
		}
		cfg.Modules[module] = struct{}{}
	}
	cfg.ID = p.ID
	cfg.Transport = p.Transport
	cfg.SASL = p.SASL
	cfg.TLS = p.TLS
	cfg.Compression = p.Compression
	cfg.ModRoster = p.ModRoster
	cfg.ModOffline = p.ModOffline
	cfg.ModRegistration = p.ModRegistration
	cfg.ModVersion = p.ModVersion
	cfg.ModPing = p.ModPing
	return nil
}

// TransportConfig represents an XMPP stream transport configuration.
type TransportConfig struct {
	Type           transport.TransportType
	BindAddress    string
	Port           int
	ConnectTimeout int
	KeepAlive      int
	MaxStanzaSize  int64
}

type transportProxyType struct {
	Type           string `yaml:"type"`
	BindAddress    string `yaml:"bind_addr"`
	Port           int    `yaml:"port"`
	ConnectTimeout int    `yaml:"connect_timeout"`
	KeepAlive      int    `yaml:"keep_alive"`
	MaxStanzaSize  int64  `yaml:"max_stanza_size"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (t *TransportConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := transportProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	// validate transport type
	switch p.Type {
	case "", "socket":
		t.Type = transport.Socket

	case "websocket":
		t.Type = transport.WebSocket

	default:
		return fmt.Errorf("server.TransportConfig: unrecognized transport type: %s", p.Type)
	}
	t.BindAddress = p.BindAddress
	t.Port = p.Port

	// assign transport's defaults
	if t.Port == 0 {
		t.Port = defaultTransportPort
	}
	t.ConnectTimeout = p.ConnectTimeout
	if t.ConnectTimeout == 0 {
		t.ConnectTimeout = defaultTransportConnectTimeout
	}
	t.KeepAlive = p.KeepAlive
	if t.KeepAlive == 0 {
		t.KeepAlive = defaultTransportKeepAlive
	}
	t.MaxStanzaSize = p.MaxStanzaSize
	if t.MaxStanzaSize == 0 {
		t.MaxStanzaSize = defaultTransportMaxStanzaSize
	}
	return nil
}

// TLSConfig represents a server TLS configuration.
type TLSConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// CompressConfig represents a server stream compression configuration.
type CompressConfig struct {
	Level compress.Level
}

type compressionProxyType struct {
	Level string `yaml:"level"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *CompressConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := compressionProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	switch p.Level {
	case "":
		c.Level = compress.NoCompression
	case "best":
		c.Level = compress.BestCompression
	case "speed":
		c.Level = compress.SpeedCompression
	case "default":
		c.Level = compress.DefaultCompression
	default:
		return fmt.Errorf("server.CompressConfig: unrecognized compression level: %s", p.Level)
	}
	return nil
}
