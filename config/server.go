/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package config

type Server struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Domains []string `yaml:"domains"`

	Transport   Transport   `yaml:"transport"`
	TLS         TLS         `yaml:"tls"`
	Compression Compression `yaml:"compression"`
	SASL        []string    `yaml:"sasl"`

	ModOffline      ModOffline      `yaml:"mod_offline"`
	ModPrivate      ModPrivate      `yaml:"mod_private"`
	ModVCard        ModVCard        `yaml:"mod_vcard"`
	ModRegistration ModRegistration `yaml:"mod_registration"`
	ModVersion      ModVersion      `yaml:"mod_version"`
	ModPing         ModPing         `yaml:"mod_ping"`
}

type Transport struct {
	Type           string `yaml:"type"`
	BindAddress    string `yaml:"bind_addr"`
	Port           int    `yaml:"port"`
	ConnectTimeout int    `yaml:"connect_timeout"`
	KeepAlive      int    `yaml:"keep_alive"`
	MaxStanzaSize  int    `yaml:"max_stanza_size"`
}

type TLS struct {
	Enabled     bool   `yaml:"enabled"`
	Required    bool   `yaml:"required"`
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

type Compression struct {
	Enabled bool   `yaml:"enabled"`
	Level   string `yaml:"level"`
}

type ModOffline struct {
	Enabled   bool `yaml:"enabled"`
	QueueSize int  `yaml:"queue_size"`
}

type ModPrivate struct {
	Enabled bool `yaml:"enabled"`
}

type ModVCard struct {
	Enabled bool `yaml:"enabled"`
}

type ModRegistration struct {
	Enabled     bool `yaml:"enabled"`
	AllowChange bool `yaml:"allow_change"`
	AllowCancel bool `yaml:"allow_cancel"`
}

type ModVersion struct {
	Enabled bool `yaml:"enabled"`
	ShowOS  bool `yaml:"show_os"`
}

type ModPing struct {
	Enabled      bool `yaml:"enabled"`
	Send         bool `yaml:"send"`
	SendInterval int  `yaml:"send_interval"`
}
