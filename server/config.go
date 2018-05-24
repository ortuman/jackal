/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/server/transport"
)

const (
	defaultTransportPort      = 5222
	defaultTransportKeepAlive = 120
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

// TransportConfig represents an XMPP stream transport configuration.
type TransportConfig struct {
	Type           transport.TransportType
	BindAddress    string
	Port           int
	ConnectTimeout int
	KeepAlive      int
	MaxStanzaSize  int
}

type transportProxyType struct {
	Type          string `yaml:"type"`
	BindAddress   string `yaml:"bind_addr"`
	Port          int    `yaml:"port"`
	KeepAlive     int    `yaml:"keep_alive"`
	MaxStanzaSize int    `yaml:"max_stanza_size"`
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
	t.KeepAlive = p.KeepAlive
	if t.KeepAlive == 0 {
		t.KeepAlive = defaultTransportKeepAlive
	}
	return nil
}

// TLSConfig represents a server TLS configuration.
type TLSConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// Config represents an XMPP server configuration.
type Config struct {
	ID        string
	Type      ServerType
	Transport TransportConfig
	TLS       TLSConfig
	C2S       c2s.Config
}

type configProxyType struct {
	ID        string          `yaml:"id"`
	Type      string          `yaml:"type"`
	Transport TransportConfig `yaml:"transport"`
	TLS       TLSConfig       `yaml:"tls"`
	C2S       c2s.Config      `yaml:"c2s"`
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
	cfg.ID = p.ID
	cfg.Transport = p.Transport
	cfg.TLS = p.TLS
	cfg.C2S = p.C2S
	return nil
}
