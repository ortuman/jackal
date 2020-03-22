/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"fmt"
	"strings"
	"time"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/transport/compress"
)

const (
	defaultConnectTimeout     = time.Duration(5) * time.Second
	defaultTimeout            = time.Duration(20) * time.Second
	defaultMaxStanzaSize      = 32768
	defaultTransportPort      = 5222
	defaultTransportKeepAlive = time.Duration(120) * time.Second
	defaultTransportURLPath   = "/xmpp/ws"
)

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

// CompressConfig represents a server Stream compression configuration.
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
		return fmt.Errorf("c2s.CompressConfig: unrecognized compression level: %s", p.Level)
	}
	return nil
}

// TransportConfig represents an XMPP stream transport configuration.
type TransportConfig struct {
	Type        transport.Type
	BindAddress string
	Port        int
	URLPath     string
}

type transportProxyType struct {
	Type        string `yaml:"type"`
	BindAddress string `yaml:"bind_addr"`
	Port        int    `yaml:"port"`
	KeepAlive   int    `yaml:"keep_alive"`
	URLPath     string `yaml:"url_path"`
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

	default:
		return fmt.Errorf("c2s.TransportConfig: unrecognized transport type: %s", p.Type)
	}
	t.BindAddress = p.BindAddress
	t.Port = p.Port

	t.URLPath = p.URLPath
	if len(t.URLPath) == 0 {
		t.URLPath = defaultTransportURLPath
	}

	// assign transport's defaults
	if t.Port == 0 {
		t.Port = defaultTransportPort
	}
	return nil
}

// TLSConfig represents a server TLS configuration.
type TLSConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// Config represents C2S server configuration.
type Config struct {
	ID               string
	ConnectTimeout   time.Duration
	Timeout          time.Duration
	KeepAlive        time.Duration
	MaxStanzaSize    int
	ResourceConflict ResourceConflictPolicy
	Transport        TransportConfig
	SASL             []string
	Compression      CompressConfig
}

type configProxy struct {
	ID               string          `yaml:"id"`
	Domain           string          `yaml:"domain"`
	TLS              TLSConfig       `yaml:"tls"`
	ConnectTimeout   int             `yaml:"connect_timeout"`
	Timeout          int             `yaml:"timeout"`
	KeepAlive        int             `yaml:"keep_alive"`
	MaxStanzaSize    int             `yaml:"max_stanza_size"`
	ResourceConflict string          `yaml:"resource_conflict"`
	Transport        TransportConfig `yaml:"transport"`
	SASL             []string        `yaml:"sasl"`
	Compression      CompressConfig  `yaml:"compression"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (cfg *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	cfg.ID = p.ID
	cfg.ConnectTimeout = time.Duration(p.ConnectTimeout) * time.Second
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	cfg.Timeout = time.Duration(p.Timeout) * time.Second
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	cfg.KeepAlive = time.Duration(p.KeepAlive) * time.Second
	if cfg.KeepAlive == 0 {
		cfg.KeepAlive = defaultTransportKeepAlive
	}
	cfg.MaxStanzaSize = p.MaxStanzaSize
	if cfg.MaxStanzaSize == 0 {
		cfg.MaxStanzaSize = defaultMaxStanzaSize
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
		return fmt.Errorf("c2s.Config: invalid resource_conflict option: %s", rc)
	}
	// validate SASL mechanisms
	for _, sasl := range p.SASL {
		switch sasl {
		case "plain", "digest_md5", "scram_sha_1", "scram_sha_256", "scram_sha_512":
			continue
		default:
			return fmt.Errorf("c2s.Config: unrecognized SASL mechanism: %s", sasl)
		}
	}
	cfg.Transport = p.Transport
	cfg.SASL = p.SASL
	cfg.Compression = p.Compression
	return nil
}

type streamConfig struct {
	connectTimeout   time.Duration
	timeout          time.Duration
	keepAlive        time.Duration
	maxStanzaSize    int
	resourceConflict ResourceConflictPolicy
	sasl             []string
	compression      CompressConfig
	onDisconnect     func(s stream.C2S)
}
