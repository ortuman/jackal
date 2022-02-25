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

package rostermodel

import "github.com/golang/protobuf/proto"

func (x *Item) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

func (x *Item) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

func (x *Items) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

func (x *Items) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

func (x *Notification) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

func (x *Notification) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

func (x *Notifications) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

func (x *Notifications) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

func (x *Groups) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

func (x *Groups) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

func (x *Version) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

func (x *Version) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}
