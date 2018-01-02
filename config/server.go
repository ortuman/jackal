/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
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

type ServerType int

const (
	// C2S represents a client to client server type.
	C2S ServerType = iota
	// S2S represents a server-to-client server type.
	S2S
)

type ChannelBindingMechanism int

const (
	TLSUnique ChannelBindingMechanism = iota
)

func (st ServerType) String() string {
	switch st {
	case C2S:
		return "c2s"
	case S2S:
		return "s2s"
	}
	return ""
}

type TransportType int

const (
	Socket TransportType = iota
)

func (tt TransportType) String() string {
	switch tt {
	case Socket:
		return "socket"
	}
	return ""
}

type CompressionLevel int

const (
	DefaultCompression CompressionLevel = iota
	BestCompression
	SpeedCompression
)

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

type Server struct {
	ID              string
	Type            ServerType
	Domains         []string
	Transport       Transport
	SASL            []string
	TLS             *TLS
	Modules         map[string]struct{}
	Compression     *Compression
	ModOffline      ModOffline
	ModRegistration ModRegistration
	ModVersion      ModVersion
	ModPing         ModPing
}

type serverProxyType struct {
	ID              string          `yaml:"id"`
	Type            string          `yaml:"type"`
	Domains         []string        `yaml:"domains"`
	Transport       Transport       `yaml:"transport"`
	SASL            []string        `yaml:"sasl"`
	TLS             *TLS            `yaml:"tls"`
	Modules         []string        `yaml:"modules"`
	Compression     *Compression    `yaml:"compression"`
	ModOffline      ModOffline      `yaml:"mod_offline"`
	ModRegistration ModRegistration `yaml:"mod_registration"`
	ModVersion      ModVersion      `yaml:"mod_version"`
	ModPing         ModPing         `yaml:"mod_ping"`
}

func (s *Server) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := serverProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	// validate server type
	switch strings.ToLower(p.Type) {
	case "c2s":
		s.Type = C2S
	case "s2s":
		s.Type = S2S
	default:
		return fmt.Errorf("config.Server: unrecognized server type: %s", p.Type)
	}
	// validate server domains
	if len(p.Domains) == 0 {
		return errors.New("config.Server: no domain specified")
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
	s.Domains = p.Domains
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

func (t *Transport) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := transportProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	// validate transport type
	switch p.Type {
	case "socket":
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

type TLS struct {
	Required    bool   `yaml:"required"`
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

type Compression struct {
	Level CompressionLevel
}

type compressionProxyType struct {
	Level string `yaml:"level"`
}

func (c *Compression) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := compressionProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	switch p.Level {
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

type ModOffline struct {
	QueueSize int `yaml:"queue_size"`
}

type ModRegistration struct {
	AllowChange bool `yaml:"allow_change"`
	AllowCancel bool `yaml:"allow_cancel"`
}

type ModVersion struct {
	ShowOS bool `yaml:"show_os"`
}

type ModPing struct {
	Send         bool `yaml:"send"`
	SendInterval int  `yaml:"send_interval"`
}
