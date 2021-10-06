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

	measuredrepository "github.com/ortuman/jackal/pkg/storage/measured"
	pgsqlrepository "github.com/ortuman/jackal/pkg/storage/pgsql"

	"github.com/ortuman/jackal/pkg/storage/repository"
)

const pgSQLRepositoryType = "pgsql"

func New(cfg Config) (repository.Repository, error) {
	if cfg.Type != pgSQLRepositoryType {
		return nil, fmt.Errorf("unrecognized repository type: %s", cfg.Type)
	}
	rep := pgsqlrepository.New(cfg.PgSQL)
	return measuredrepository.New(rep), nil
}
