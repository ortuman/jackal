/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"crypto/tls"
	"time"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xmpp"
	"github.com/pkg/errors"
)

const (
	defaultTransportPort      = 5269
	defaultTransportKeepAlive = time.Duration(10) * time.Minute
	defaultDialTimeout        = time.Duration(15) * time.Second
	defaultConnectTimeout     = time.Duration(5) * time.Second
	defaultTimeout            = time.Duration(20) * time.Second
	defaultMaxStanzaSize      = 131072
)

// TransportConfig represents s2s transport configuration.
type TransportConfig struct {
	BindAddress string
	Port        int
	KeepAlive   time.Duration
}

type transportConfigProxy struct {
	BindAddress string `yaml:"bind_addr"`
	Port        int    `yaml:"port"`
	KeepAlive   int    `yaml:"keep_alive"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *TransportConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := transportConfigProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.BindAddress = p.BindAddress
	c.Port = p.Port
	if c.Port == 0 {
		c.Port = defaultTransportPort
	}
	if p.KeepAlive > 0 {
		c.KeepAlive = time.Duration(p.KeepAlive) * time.Second
	} else {
		c.KeepAlive = defaultTransportKeepAlive
	}
	return nil
}

// TLSConfig represents a server TLS configuration.
type TLSConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// Config represents an s2s configuration.
type Config struct {
	ID             string
	DialTimeout    time.Duration
	ConnectTimeout time.Duration
	Timeout        time.Duration
	DialbackSecret string
	MaxStanzaSize  int
	Transport      TransportConfig
}

type configProxy struct {
	ID             string          `yaml:"id"`
	DialTimeout    int             `yaml:"dial_timeout"`
	ConnectTimeout int             `yaml:"connect_timeout"`
	Timeout        int             `yaml:"timeout"`
	DialbackSecret string          `yaml:"dialback_secret"`
	MaxStanzaSize  int             `yaml:"max_stanza_size"`
	Transport      TransportConfig `yaml:"transport"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.ID = p.ID
	c.DialbackSecret = p.DialbackSecret
	if len(c.DialbackSecret) == 0 {
		return errors.New("s2s.Config: must specify a dialback secret")
	}
	c.DialTimeout = time.Duration(p.DialTimeout) * time.Second
	if c.DialTimeout == 0 {
		c.DialTimeout = defaultDialTimeout
	}
	c.ConnectTimeout = time.Duration(p.ConnectTimeout) * time.Second
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = defaultConnectTimeout
	}
	c.Timeout = time.Duration(p.Timeout) * time.Second
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
	c.Transport = p.Transport
	c.MaxStanzaSize = p.MaxStanzaSize
	if c.MaxStanzaSize == 0 {
		c.MaxStanzaSize = defaultMaxStanzaSize
	}
	return nil
}

type streamConfig struct {
	keyGen          *keyGen
	localDomain     string
	remoteDomain    string
	connectTimeout  time.Duration
	timeout         time.Duration
	tls             *tls.Config
	transport       transport.Transport
	maxStanzaSize   int
	dbVerify        xmpp.XElement
	dialer          *dialer
	onInDisconnect  func(s stream.S2SIn)
	onOutDisconnect func(s stream.S2SOut)
}

type outStreamConfig struct {
	keyGen          *keyGen
	localDomain     string
	remoteDomain    string
	connectTimeout  time.Duration
	timeout         time.Duration
	tls             *tls.Config
	transport       transport.Transport
	maxStanzaSize   int
	dbVerify        xmpp.XElement
	onOutDisconnect func(s stream.S2SOut)
}
