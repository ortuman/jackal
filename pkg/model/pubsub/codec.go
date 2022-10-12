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

package pubsubmodel

import "google.golang.org/protobuf/proto"

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Node) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Node) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Nodes) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Nodes) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Affiliation) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Affiliation) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Affiliations) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Affiliations) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Subscription) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Subscription) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Subscriptions) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Subscriptions) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Item) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Item) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}

// MarshalBinary satisfies encoding.BinaryMarshaler interface.
func (x *Items) MarshalBinary() (data []byte, err error) {
	return proto.Marshal(x)
}

// UnmarshalBinary satisfies encoding.BinaryUnmarshaler interface.
func (x *Items) UnmarshalBinary(data []byte) error {
	return proto.Unmarshal(data, x)
}
