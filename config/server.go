/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package config

import (
	"errors"
	"fmt"
	"strings"
)

const defaultTransportPort = 5222

const defaultTransportBufferSize = 4096

const defaultTransportConnectTimeout = 5
const defaultTransportKeepAlive = 120

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

// ChannelBindingMechanism represents a scram channel binding mechanism.
type ChannelBindingMechanism int

const (
	// TLSUnique represents 'tls-unique' channel binding mechanism.
	TLSUnique ChannelBindingMechanism = iota
)

// TransportType represents a stream transport type (socket).
type TransportType int

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

const (
	// Socket represents a socket transport.
	Socket TransportType = iota
)

// String returns TransportType string representation.
func (tt TransportType) String() string {
	switch tt {
	case Socket:
		return "socket"
	}
	return ""
}

// CompressionLevel represents a stream compression level.
type CompressionLevel int

const (
	// NoCompression represents no stream compression.
	NoCompression CompressionLevel = iota

	// DefaultCompression represents 'default' stream compression level.
	DefaultCompression

	// BestCompression represents 'best for size' stream compression level.
	BestCompression

	// SpeedCompression represents 'best for speed' stream compression level.
	SpeedCompression
)

// String returns CompressionLevel string representation.
func (cl CompressionLevel) String() string {
	switch cl {
	case DefaultCompression:
		return "default"
	case BestCompression:
		return "best"
	case SpeedCompression:
		return "speed"
	}
	return ""
}

// Server represents an XMPP server configuration.
type Server struct {
	ID               string
	Type             ServerType
	ResourceConflict ResourceConflictPolicy
	Transport        Transport
	SASL             []string
	TLS              TLS
	Modules          map[string]struct{}
	Compression      Compression
	ModOffline       ModOffline
	ModRegistration  ModRegistration
	ModVersion       ModVersion
	ModPing          ModPing
}

type serverProxyType struct {
	ID               string          `yaml:"id"`
	Type             string          `yaml:"type"`
	ResourceConflict string          `yaml:"resource_conflict"`
	Transport        Transport       `yaml:"transport"`
	SASL             []string        `yaml:"sasl"`
	TLS              TLS             `yaml:"tls"`
	Modules          []string        `yaml:"modules"`
	Compression      Compression     `yaml:"compression"`
	ModOffline       ModOffline      `yaml:"mod_offline"`
	ModRegistration  ModRegistration `yaml:"mod_registration"`
	ModVersion       ModVersion      `yaml:"mod_version"`
	ModPing          ModPing         `yaml:"mod_ping"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (s *Server) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := serverProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	// validate server type
	switch strings.ToLower(p.Type) {
	case "c2s":
		s.Type = C2SServerType
	case "s2s":
		return errors.New("config.Server: s2s server type not yet supported")
	default:
		return fmt.Errorf("config.Server: unrecognized server type: %s", p.Type)
	}
	// validate resource conflict policy type
	rc := strings.ToLower(p.ResourceConflict)
	switch rc {
	case "override":
		s.ResourceConflict = Override
	case "reject":
		s.ResourceConflict = Reject
	case "", "replace":
		s.ResourceConflict = Replace
	default:
		return fmt.Errorf("invalid resource_conflict option: %s", rc)
	}
	// validate SASL mechanisms
	for _, sasl := range p.SASL {
		switch sasl {
		case "plain", "digest_md5", "scram_sha_1", "scram_sha_256":
			continue
		default:
			return fmt.Errorf("config.Server: unrecognized SASL mechanism: %s", sasl)
		}
	}
	// validate modules
	s.Modules = map[string]struct{}{}
	for _, module := range p.Modules {
		switch module {
		case "roster", "private", "vcard", "registration", "version", "ping", "offline":
			break
		default:
			return fmt.Errorf("config.Server: unrecognized module: %s", module)
		}
		s.Modules[module] = struct{}{}
	}
	s.ID = p.ID
	s.Transport = p.Transport
	s.SASL = p.SASL
	s.TLS = p.TLS
	s.Compression = p.Compression
	s.ModOffline = p.ModOffline
	s.ModRegistration = p.ModRegistration
	s.ModVersion = p.ModVersion
	s.ModPing = p.ModPing
	return nil
}

// Transport represents an XMPP stream transport configuration.
type Transport struct {
	Type           TransportType
	BindAddress    string
	Port           int
	ConnectTimeout int
	KeepAlive      int
	BufferSize     int
}

type transportProxyType struct {
	Type           string `yaml:"type"`
	BindAddress    string `yaml:"bind_addr"`
	Port           int    `yaml:"port"`
	ConnectTimeout int    `yaml:"connect_timeout"`
	KeepAlive      int    `yaml:"keep_alive"`
	MaxStanzaSize  int    `yaml:"max_stanza_size"`
	BufferSize     int    `yaml:"buf_size"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (t *Transport) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := transportProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	// validate transport type
	switch p.Type {
	case "", "socket":
		t.Type = Socket
	default:
		return fmt.Errorf("config.Transport: unrecognized transport type: %s", p.Type)
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
	t.BufferSize = p.BufferSize
	if t.BufferSize == 0 {
		t.BufferSize = defaultTransportBufferSize
	}
	return nil
}

// TLS represents a server TLS configuration.
type TLS struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// Compression represents a server stream compression configuration.
type Compression struct {
	Level CompressionLevel
}

type compressionProxyType struct {
	Level string `yaml:"level"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Compression) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := compressionProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	switch p.Level {
	case "":
		c.Level = NoCompression
	case "best":
		c.Level = BestCompression
	case "speed":
		c.Level = SpeedCompression
	case "default":
		c.Level = DefaultCompression
	default:
		return fmt.Errorf("config.Compress: unrecognized compression level: %s", p.Level)
	}
	return nil
}

// ModOffline represents Offline Storage module configuration.
type ModOffline struct {
	QueueSize int `yaml:"queue_size"`
}

// ModRegistration represents XMPP In-Band Registration module (XEP-0077) configuration.
type ModRegistration struct {
	AllowRegistration bool `yaml:"allow_registration"`
	AllowChange       bool `yaml:"allow_change"`
	AllowCancel       bool `yaml:"allow_cancel"`
}

// ModVersion represents XMPP Software Version module (XEP-0092) configuration.
type ModVersion struct {
	ShowOS bool `yaml:"show_os"`
}

// ModPing represents XMPP Ping module (XEP-0199) configuration.
type ModPing struct {
	Send         bool `yaml:"send"`
	SendInterval int  `yaml:"send_interval"`
}
