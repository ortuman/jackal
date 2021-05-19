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

package offline

import (
	"context"

	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"

	"github.com/ortuman/jackal/pkg/cluster/locker"
	"github.com/ortuman/jackal/pkg/repository"
	"github.com/ortuman/jackal/pkg/router"
)

//go:generate moq -out repository.mock_test.go . offlineRepository:repositoryMock
type offlineRepository interface {
	repository.Offline
}

//go:generate moq -out router.mock_test.go . globalRouter:routerMock
type globalRouter interface {
	router.Router
}

//go:generate moq -out hosts.mock_test.go . hosts
type hosts interface {
	IsLocalHost(h string) bool
}

//go:generate moq -out locker.mock_test.go . clusterLocker:lockerMock
type clusterLocker interface {
	locker.Locker
}

//go:generate moq -out lock.mock_test.go . clusterLock:lockMock
type clusterLock interface {
	locker.Lock
}

//go:generate moq -out resourcemanager.mock_test.go . resourceManager
type resourceManager interface {
	GetResources(ctx context.Context, username string) ([]c2smodel.Resource, error)
}
