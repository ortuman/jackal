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

package storage

import (
	"fmt"

	kitlog "github.com/go-kit/log"
	"github.com/ortuman/jackal/pkg/storage/boltdb"
	cachedrepository "github.com/ortuman/jackal/pkg/storage/cached"
	measuredrepository "github.com/ortuman/jackal/pkg/storage/measured"
	pgsqlrepository "github.com/ortuman/jackal/pkg/storage/pgsql"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const (
	boltDBRepositoryType = "boltdb"
	pgSQLRepositoryType  = "pgsql"
)

// Config contains generic storage configuration.
type Config struct {
	Type   string                  `fig:"type" default:"boltdb"`
	PgSQL  pgsqlrepository.Config  `fig:"pgsql"`
	BoltDB boltdb.Config           `fig:"boltdb"`
	Cache  cachedrepository.Config `fig:"cache"`
}

// New returns an initialized repository.Repository derived from cfg configuration.
func New(cfg Config, logger kitlog.Logger) (repository.Repository, error) {
	var rep repository.Repository

	switch cfg.Type {
	case pgSQLRepositoryType:
		rep = pgsqlrepository.New(cfg.PgSQL, logger)

	case boltDBRepositoryType:
		rep = boltdb.New(cfg.BoltDB, logger)

	default:
		return nil, fmt.Errorf("unrecognized repository type: %s", cfg.Type)
	}

	if len(cfg.Cache.Type) > 0 {
		var err error
		rep, err = cachedrepository.New(cfg.Cache, rep, logger)
		if err != nil {
			return nil, err
		}
	}
	return measuredrepository.New(rep), nil
}
