// Copyright 2022 The jackal Authors
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

package jackal

import (
	"path/filepath"

	"github.com/ortuman/jackal/pkg/module/xep0313"

	"github.com/kkyr/fig"
	adminserver "github.com/ortuman/jackal/pkg/admin/server"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/c2s"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	clusterserver "github.com/ortuman/jackal/pkg/cluster/server"
	"github.com/ortuman/jackal/pkg/component/xep0114"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/module/offline"
	"github.com/ortuman/jackal/pkg/module/xep0092"
	"github.com/ortuman/jackal/pkg/module/xep0198"
	"github.com/ortuman/jackal/pkg/module/xep0199"
	"github.com/ortuman/jackal/pkg/s2s"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/storage"
)

const (
	kvClusterType   = "kv"
	noneClusterType = "none"
)

// LoggerConfig defines logger configuration.
type LoggerConfig struct {
	Level  string `fig:"level" default:"debug"`
	Format string `fig:"format"`
}

// HTTPConfig defines HTTP configuration.
type HTTPConfig struct {
	Port int `fig:"port" default:"6060"`
}

// ClusterConfig defines cluster configuration.
type ClusterConfig struct {
	Type   string               `fig:"type" default:"none"`
	KV     kv.Config            `fig:"kv"`
	Server clusterserver.Config `fig:"server"`
}

// IsEnabled tells whether cluster config is enabled.
func (c ClusterConfig) IsEnabled() bool {
	return c.Type != noneClusterType
}

// C2SConfig defines C2S subsystem configuration.
type C2SConfig struct {
	Listeners c2s.ListenersConfig `fig:"listeners"`
}

// S2SConfig defines S2S subsystem configuration.
type S2SConfig struct {
	Listeners s2s.ListenersConfig `fig:"listeners"`
	Out       s2s.OutConfig       `fig:"out"`
}

// ComponentsConfig defines application components configuration.
type ComponentsConfig struct {
	Listeners xep0114.ListenersConfig `fig:"listeners"`
	Secret    string                  `fig:"secret"`
}

// ModulesConfig defines application modules configuration.
type ModulesConfig struct {
	// Enabled specifies total set of enabled modules
	Enabled []string `fig:"enabled"`

	// Offline: offline storage
	Offline offline.Config `fig:"offline"`

	// XEP-0092: Software Version
	Version xep0092.Config `fig:"version"`

	// XEP-0198: Stream Management
	Stream xep0198.Config `fig:"stream"`

	// XEP-0199: XMPP Ping
	Ping xep0199.Config `fig:"ping"`

	// XEP-0313: Message Archive Management
	Mam xep0313.Config `fig:"mam"`
}

// Config defines jackal application configuration.
type Config struct {
	MemoryBallastSize int `fig:"memory_ballast_size" default:"134217728"`

	Logger  LoggerConfig  `fig:"logger"`
	Cluster ClusterConfig `fig:"cluster"`

	HTTP HTTPConfig `fig:"http"`

	Peppers pepper.Config      `fig:"peppers"`
	Admin   adminserver.Config `fig:"admin"`
	Storage storage.Config     `fig:"storage"`
	Hosts   host.Configs       `fig:"hosts"`
	Shapers []shaper.Config    `fig:"shapers"`

	C2S        C2SConfig        `fig:"c2s"`
	S2S        S2SConfig        `fig:"s2s"`
	Components ComponentsConfig `fig:"components"`
	Modules    ModulesConfig    `fig:"modules"`
}

func loadConfig(configFile string) (*Config, error) {
	var cfg Config
	file := filepath.Base(configFile)
	dir := filepath.Dir(configFile)

	err := fig.Load(&cfg, fig.File(file), fig.Dirs(dir), fig.UseEnv("jackal"))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
