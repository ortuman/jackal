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

package xep0012

import (
	"context"

	coremodel "github.com/ortuman/jackal/model/core"
	"github.com/ortuman/jackal/repository"
)

//go:generate moq -out repository.mock_test.go . globalRepository:repositoryMock
type globalRepository interface {
	repository.Repository
}

//go:generate moq -out resource_manager.mock_test.go . resourceManager
type resourceManager interface {
	GetResources(ctx context.Context, username string) ([]coremodel.Resource, error)
}
