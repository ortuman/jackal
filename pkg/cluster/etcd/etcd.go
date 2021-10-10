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

package etcd

import "time"

type Config struct {
	Endpoints            []string      `fig:"endpoints" default:"[http://localhost:2379]"`
	DialTimeout          time.Duration `fig:"dial_timeout" default:"20s"`
	DialKeepAliveTime    time.Duration `fig:"dial_keep_alive_time" default:"30s"`
	DialKeepAliveTimeout time.Duration `fig:"dial_keep_alive_timeout" default:"10s"`
	KeepAliveTime        time.Duration `fig:"keep_alive" default:"10s"`
	Timeout              time.Duration `fig:"keep_alive" default:"20m"`
}
