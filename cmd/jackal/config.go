// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"path/filepath"
	"time"

	"github.com/ortuman/jackal/pkg/storage"

	"github.com/kkyr/fig"
)

type peppersConfig struct {
	Keys  map[string]string `fig:"keys"`
	UseID string            `fig:"use"`
}

type loggerConfig struct {
	Level      string `fig:"level" default:"debug"`
	OutputPath string `fig:"output_path"`
}

type adminConfig struct {
	BindAddr string `fig:"bind_addr"`
	Port     int    `fig:"port" default:"15280"`
	Disabled bool   `fig:"disabled"`
}

type etcdConfig struct {
	Endpoints   []string      `fig:"endpoints" default:"[http://localhost:2379]"`
	DialTimeout time.Duration `fig:"dial_timeout" default:"5s"`
}

type clusterConfig struct {
	Etcd     etcdConfig `fig:"etcd"`
	BindAddr string     `fig:"bind_addr"`
	Port     int        `fig:"port" default:"14369"`
}

type shaperConfig struct {
	Name        string `fig:"name"`
	MaxSessions int    `fig:"max_sessions" default:"10"`
	Rate        struct {
		Limit int `fig:"limit" default:"1000"`
		Burst int `fig:"burst" default:"0"`
	} `fig:"rate"`
	Matching struct {
		JID struct {
			In    []string `fig:"in"`
			RegEx string   `fig:"regex"`
		}
	} `fig:"matching"`
}

type hostConfig struct {
	Domain string `fig:"domain"`
	TLS    struct {
		CertFile       string `fig:"cert_file"`
		PrivateKeyFile string `fig:"privkey_file"`
	} `fig:"tls"`
}

type listenerConfig struct {
	Type      string `fig:"type" default:"c2s"`
	BindAddr  string `fig:"bind_addr"`
	Port      int    `fig:"port" default:"5222"`
	Transport string `fig:"transport" default:"socket"`
	DirectTLS bool   `fig:"direct_tls"`
	SASL      struct {
		Mechanisms []string `fig:"mechanisms" default:"[scram_sha_1, scram_sha_256, scram_sha_512, scram_sha3_512]"`
		External   struct {
			Address  string `fig:"address"`
			IsSecure bool   `fig:"is_secure"`
		} `fig:"external"`
	} `fig:"sasl"`
	CompressionLevel    string        `fig:"compression_level" default:"default"`
	ResourceConflict    string        `fig:"resource_conflict" default:"terminate_old"`
	MaxStanzaSize       int           `fig:"max_stanza_size" default:"32768"`
	Secret              string        `fig:"secret"`
	ConnectTimeout      time.Duration `fig:"conn_timeout" default:"3s"`
	AuthenticateTimeout time.Duration `fig:"auth_timeout" default:"10s"`
	KeepAliveTimeout    time.Duration `fig:"keep_alive_timeout" default:"10m"`
	RequestTimeout      time.Duration `fig:"req_timeout" default:"15s"`
}

type s2sOutConfig struct {
	DialTimeout      time.Duration `fig:"dial_timeout" default:"5s"`
	DialbackSecret   string        `fig:"secret"`
	ConnectTimeout   time.Duration `fig:"conn_timeout" default:"3s"`
	KeepAliveTimeout time.Duration `fig:"keep_alive_timeout" default:"120s"`
	RequestTimeout   time.Duration `fig:"req_timeout" default:"15s"`
	MaxStanzaSize    int           `fig:"max_stanza_size" default:"131072"`
}

type modulesConfig struct {
	Enabled []string `fig:"enabled"`

	Offline struct {
		QueueSize int `fig:"queue_size" default:"200"`
	} `fig:"offline"`

	// XEP-0092: Software Version
	Version struct {
		ShowOS bool `fig:"show_os"`
	} `fig:"version"`

	// XEP-0198: Stream Management
	Stream struct {
		HibernateTime      time.Duration `fig:"hibernate_time" default:"3m"`
		RequestAckInterval time.Duration `fig:"request_ack_interval" default:"2m"`
		WaitForAckTimeout  time.Duration `fig:"wait_for_ack_timeout" default:"30s"`
		MaxQueueSize       int           `fig:"max_queue_size" default:"30"`
	}

	// XEP-0199: XMPP Ping
	Ping struct {
		AckTimeout    time.Duration `fig:"ack_timeout" default:"32s"`
		Interval      time.Duration `fig:"interval" default:"1m"`
		SendPings     bool          `fig:"send_pings"`
		TimeoutAction string        `fig:"timeout_action" default:"none"`
	} `fig:"ping"`
}

type componentsConfig struct {
}

type serverConfig struct {
	HTTPPort   int              `fig:"http_port" default:"6060"`
	Peppers    peppersConfig    `fig:"peppers"`
	Logger     loggerConfig     `fig:"logger"`
	Admin      adminConfig      `fig:"admin"`
	Cluster    clusterConfig    `fig:"cluster"`
	Storage    storage.Config   `fig:"storage"`
	Hosts      []hostConfig     `fig:"hosts"`
	Listeners  []listenerConfig `fig:"listeners"`
	Shapers    []shaperConfig   `fig:"shapers"`
	S2SOut     s2sOutConfig     `fig:"s2s_out"`
	Modules    modulesConfig    `fig:"modules"`
	Components componentsConfig `fig:"components"`
}

func loadConfig(configFile string) (*serverConfig, error) {
	var cfg serverConfig
	file := filepath.Base(configFile)
	dir := filepath.Dir(configFile)

	err := fig.Load(&cfg, fig.File(file), fig.Dirs(dir))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
