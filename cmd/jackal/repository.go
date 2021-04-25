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
	measuredrepository "github.com/ortuman/jackal/repository/measured"
	pgsqlrepository "github.com/ortuman/jackal/repository/pgsql"
)

func initRepository(a *serverApp, sCfg storageConfig) error {
	cfg := sCfg.PgSQL
	opts := pgsqlrepository.Config{
		MaxIdleConns:    cfg.MaxIdleConns,
		MaxOpenConns:    cfg.MaxOpenConns,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}
	pgRep := pgsqlrepository.New(
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
		opts,
	)
	a.rep = measuredrepository.New(pgRep)
	a.registerStartStopper(a.rep)
	return nil
}
