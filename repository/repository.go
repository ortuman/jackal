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

package repository

import "context"

// Repository represents application repository interface.
type Repository interface {
	baseRepository

	// InTransaction generates a repository transaction and completes it after it's being used by f function.
	// In case f returns no error tx transaction will be committed.
	InTransaction(ctx context.Context, f func(ctx context.Context, tx Transaction) error) error

	// Start initializes repository.
	Start(ctx context.Context) error

	// Stop releases all underlying repository resources.
	Stop(ctx context.Context) error
}

// Transaction represents a repository transaction interface.
// All repository supported operations must be allowed to be used through a derived transaction.
type Transaction interface {
	baseRepository
}

type baseRepository interface {
	User
	Capabilities
	Offline
	BlockList
	Private
	Roster
	VCard
}
