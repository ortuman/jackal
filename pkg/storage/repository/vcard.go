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

package repository

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"
)

// VCard defines vCard repository operations.
type VCard interface {
	// UpsertVCard inserts a new vCard element into repository.
	UpsertVCard(ctx context.Context, vCard stravaganza.Element, username string) error

	// FetchVCard retrieves from repository a user vCard.
	FetchVCard(ctx context.Context, username string) (stravaganza.Element, error)

	// DeleteVCard deletes a vCard entity from repository.
	DeleteVCard(ctx context.Context, username string) error
}
