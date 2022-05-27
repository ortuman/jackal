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

package pgsqlrepository

import (
	"context"
	"time"
)

const waitForLockDelay = time.Millisecond * 10

type pgSQLLocker struct {
	conn conn
}

func (l *pgSQLLocker) Lock(ctx context.Context, lockID string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var acquired bool

		err := l.conn.QueryRowContext(ctx, "/*NO LOAD BALANCE*/ SELECT pg_try_advisory_lock(hashtext($1))", lockID).Scan(&acquired)
		switch err {
		case nil:
			if acquired {
				return nil
			}
			time.Sleep(waitForLockDelay) // wait and retry

		default:
			return err
		}
	}
}

func (l *pgSQLLocker) Unlock(ctx context.Context, lockID string) error {
	_, err := l.conn.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext($1))", lockID)
	return err
}
