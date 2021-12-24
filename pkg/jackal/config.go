// Copyright 2021 The jackal Authors
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

	"github.com/kkyr/fig"
	adminserver "github.com/ortuman/jackal/pkg/admin/server"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/c2s"
	"github.com/ortuman/jackal/pkg/cluster/etcd"
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

type LoggerConfig struct {
	Level      string `fig:"level" default:"debug"`
	OutputPath string `fig:"output_path"`
}

type ClusterConfig struct {
	Etcd   etcd.Config          `fig:"etcd"`
	Server clusterserver.Config `fig:"server"`
}

type C2SConfig struct {
	Listeners c2s.ListenersConfig `fig:"listeners"`
}

type S2SConfig struct {
	Listeners s2s.ListenersConfig `fig:"listeners"`
	Out       s2s.OutConfig       `fig:"out"`
}

type ComponentsConfig struct {
	Listeners xep0114.ListenersConfig `fig:"listeners"`
}

type ModulesConfig struct {
	// Enabled defines total set of enabled modules
	Enabled []string `fig:"enabled"`

	// Offline offline storage
	Offline offline.Config `fig:"offline"`

	// XEP-0092: Software Version
	Version xep0092.Config `fig:"version"`

	// XEP-0198: Stream Management
	Stream xep0198.Config `fig:"stream"`

	// XEP-0199: XMPP Ping
	Ping xep0199.Config `fig:"ping"`
}

type Config struct {
	Logger  LoggerConfig  `fig:"logger"`
	Cluster ClusterConfig `fig:"cluster"`

	HTTPPort int `fig:"http_port" default:"6060"`

	Peppers pepper.Config      `fig:"peppers"`
	Admin   adminserver.Config `fig:"admin"`
	Storage storage.Config     `fig:"storage"`
	Hosts   []host.Config      `fig:"hosts"`
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

	err := fig.Load(&cfg, fig.File(file), fig.Dirs(dir))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
