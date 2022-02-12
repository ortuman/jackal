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

package memberlist

import (
	"context"

	clustermodel "github.com/ortuman/jackal/pkg/model/cluster"
)

// NewNop returns a memberlist that doesn't do anything.
func NewNop() MemberList {
	return &nopMemberList{
		members: make(map[string]clustermodel.Member),
	}
}

type nopMemberList struct {
	members map[string]clustermodel.Member
}

func (ml *nopMemberList) GetMember(instanceID string) (m clustermodel.Member, ok bool) {
	m, ok = ml.members[instanceID]
	return
}

func (ml *nopMemberList) GetMembers() map[string]clustermodel.Member {
	return ml.members
}

func (ml *nopMemberList) Start(_ context.Context) error {
	return nil
}

func (ml *nopMemberList) Stop(_ context.Context) error {
	return nil
}
