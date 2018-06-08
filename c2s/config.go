/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"fmt"
	"strings"

	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/transport/compress"
)

const (
	defaultTransportConnectTimeout = 5
	defaultTransportMaxStanzaSize  = 32768
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

// ModulesConfig represents C2S modules configuration.
type ModulesConfig struct {
	Enabled      map[string]struct{}
	Roster       roster.Config
	Offline      offline.Config
	Registration xep0077.Config
	Version      xep0092.Config
	Ping         xep0199.Config
}

type modulesConfigProxy struct {
	Enabled      []string       `yaml:"enabled"`
	Roster       roster.Config  `yaml:"mod_roster"`
	Offline      offline.Config `yaml:"mod_offline"`
	Registration xep0077.Config `yaml:"mod_registration"`
	Version      xep0092.Config `yaml:"mod_version"`
	Ping         xep0199.Config `yaml:"mod_ping"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (cfg *ModulesConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := modulesConfigProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	// validate modules
	enabled := make(map[string]struct{}, len(p.Enabled))
	for _, mod := range p.Enabled {
		switch mod {
		case "roster", "last_activity", "private", "vcard", "registration", "version", "blocking_command",
			"ping", "offline":
			break
		default:
			return fmt.Errorf("c2s.ModulesConfig: unrecognized module: %s", mod)
		}
		enabled[mod] = struct{}{}
	}
	cfg.Enabled = enabled
	cfg.Roster = p.Roster
	cfg.Offline = p.Offline
	cfg.Registration = p.Registration
	cfg.Version = p.Version
	cfg.Ping = p.Ping
	return nil
}

// Config represents C2S Stream configuration.
type Config struct {
	ConnectTimeout   int
	MaxStanzaSize    int
	ResourceConflict ResourceConflictPolicy
	SASL             []string
	Compression      CompressConfig
	Modules          ModulesConfig
}

type configProxy struct {
	ConnectTimeout   int            `yaml:"connect_timeout"`
	MaxStanzaSize    int            `yaml:"max_stanza_size"`
	ResourceConflict string         `yaml:"resource_conflict"`
	SASL             []string       `yaml:"sasl"`
	Compression      CompressConfig `yaml:"compression"`
	Modules          ModulesConfig  `yaml:"modules"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (cfg *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
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
		case "plain", "digest_md5", "scram_sha_1", "scram_sha_256":
			continue
		default:
			return fmt.Errorf("c2s.Config: unrecognized SASL mechanism: %s", sasl)
		}
	}
	cfg.ConnectTimeout = p.ConnectTimeout
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = defaultTransportConnectTimeout
	}
	cfg.MaxStanzaSize = p.MaxStanzaSize
	if cfg.MaxStanzaSize == 0 {
		cfg.MaxStanzaSize = defaultTransportMaxStanzaSize
	}
	cfg.SASL = p.SASL
	cfg.Compression = p.Compression
	cfg.Modules = p.Modules
	return nil
}
