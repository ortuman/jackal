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

package resourcemanager

import (
	"github.com/ortuman/jackal/cluster/kv"
	"github.com/ortuman/jackal/model"
)

//go:generate moq -out kv.mock_test.go . kvStorage:kvMock
type kvStorage interface {
	kv.KV
}

//go:generate moq -out memberlist.mock_test.go . memberList
type memberList interface {
	GetMember(instanceID string) (m model.ClusterMember, ok bool)
}
