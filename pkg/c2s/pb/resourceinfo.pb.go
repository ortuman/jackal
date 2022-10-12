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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.9
// source: proto/c2s/v1/resourceinfo.proto

package pb

import (
	stravaganza "github.com/jackal-xmpp/stravaganza"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// ResourceInfo represents resource associated info content.
type ResourceInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// instance_id indicates what instance registered this resource.
	InstanceId string `protobuf:"bytes,1,opt,name=instance_id,json=instanceId,proto3" json:"instance_id,omitempty"`
	// domain is the resource associated domain.
	Domain string `protobuf:"bytes,2,opt,name=domain,proto3" json:"domain,omitempty"`
	// info is the resource additional context info.
	Info map[string]string `protobuf:"bytes,3,rep,name=info,proto3" json:"info,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// presence is the resource last received presence.
	Presence *stravaganza.PBElement `protobuf:"bytes,4,opt,name=presence,proto3" json:"presence,omitempty"`
}

func (x *ResourceInfo) Reset() {
	*x = ResourceInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_c2s_v1_resourceinfo_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResourceInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceInfo) ProtoMessage() {}

func (x *ResourceInfo) ProtoReflect() protoreflect.Message {
	mi := &file_proto_c2s_v1_resourceinfo_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceInfo.ProtoReflect.Descriptor instead.
func (*ResourceInfo) Descriptor() ([]byte, []int) {
	return file_proto_c2s_v1_resourceinfo_proto_rawDescGZIP(), []int{0}
}

func (x *ResourceInfo) GetInstanceId() string {
	if x != nil {
		return x.InstanceId
	}
	return ""
}

func (x *ResourceInfo) GetDomain() string {
	if x != nil {
		return x.Domain
	}
	return ""
}

func (x *ResourceInfo) GetInfo() map[string]string {
	if x != nil {
		return x.Info
	}
	return nil
}

func (x *ResourceInfo) GetPresence() *stravaganza.PBElement {
	if x != nil {
		return x.Presence
	}
	return nil
}

var File_proto_c2s_v1_resourceinfo_proto protoreflect.FileDescriptor

var file_proto_c2s_v1_resourceinfo_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x32, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x72,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x69, 0x6e, 0x66, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x06, 0x63, 0x32, 0x73, 0x2e, 0x76, 0x31, 0x1a, 0x34, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x61, 0x63, 0x6b, 0x61, 0x6c, 0x2d, 0x78, 0x6d, 0x70,
	0x70, 0x2f, 0x73, 0x74, 0x72, 0x61, 0x76, 0x61, 0x67, 0x61, 0x6e, 0x7a, 0x61, 0x2f, 0x73, 0x74,
	0x72, 0x61, 0x76, 0x61, 0x67, 0x61, 0x6e, 0x7a, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xe8, 0x01, 0x0a, 0x0c, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x6e, 0x66, 0x6f,
	0x12, 0x1f, 0x0a, 0x0b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x49,
	0x64, 0x12, 0x16, 0x0a, 0x06, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x12, 0x32, 0x0a, 0x04, 0x69, 0x6e, 0x66,
	0x6f, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x63, 0x32, 0x73, 0x2e, 0x76, 0x31,
	0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x2e, 0x49, 0x6e,
	0x66, 0x6f, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x69, 0x6e, 0x66, 0x6f, 0x12, 0x32, 0x0a,
	0x08, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x16, 0x2e, 0x73, 0x74, 0x72, 0x61, 0x76, 0x61, 0x67, 0x61, 0x6e, 0x7a, 0x61, 0x2e, 0x50, 0x42,
	0x45, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x52, 0x08, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x63,
	0x65, 0x1a, 0x37, 0x0a, 0x09, 0x49, 0x6e, 0x66, 0x6f, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x0c, 0x5a, 0x0a, 0x70, 0x6b,
	0x67, 0x2f, 0x63, 0x32, 0x73, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_c2s_v1_resourceinfo_proto_rawDescOnce sync.Once
	file_proto_c2s_v1_resourceinfo_proto_rawDescData = file_proto_c2s_v1_resourceinfo_proto_rawDesc
)

func file_proto_c2s_v1_resourceinfo_proto_rawDescGZIP() []byte {
	file_proto_c2s_v1_resourceinfo_proto_rawDescOnce.Do(func() {
		file_proto_c2s_v1_resourceinfo_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_c2s_v1_resourceinfo_proto_rawDescData)
	})
	return file_proto_c2s_v1_resourceinfo_proto_rawDescData
}

var file_proto_c2s_v1_resourceinfo_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proto_c2s_v1_resourceinfo_proto_goTypes = []interface{}{
	(*ResourceInfo)(nil),          // 0: c2s.v1.ResourceInfo
	nil,                           // 1: c2s.v1.ResourceInfo.InfoEntry
	(*stravaganza.PBElement)(nil), // 2: stravaganza.PBElement
}
var file_proto_c2s_v1_resourceinfo_proto_depIdxs = []int32{
	1, // 0: c2s.v1.ResourceInfo.info:type_name -> c2s.v1.ResourceInfo.InfoEntry
	2, // 1: c2s.v1.ResourceInfo.presence:type_name -> stravaganza.PBElement
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proto_c2s_v1_resourceinfo_proto_init() }
func file_proto_c2s_v1_resourceinfo_proto_init() {
	if File_proto_c2s_v1_resourceinfo_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_c2s_v1_resourceinfo_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResourceInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_c2s_v1_resourceinfo_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_c2s_v1_resourceinfo_proto_goTypes,
		DependencyIndexes: file_proto_c2s_v1_resourceinfo_proto_depIdxs,
		MessageInfos:      file_proto_c2s_v1_resourceinfo_proto_msgTypes,
	}.Build()
	File_proto_c2s_v1_resourceinfo_proto = out.File
	file_proto_c2s_v1_resourceinfo_proto_rawDesc = nil
	file_proto_c2s_v1_resourceinfo_proto_goTypes = nil
	file_proto_c2s_v1_resourceinfo_proto_depIdxs = nil
}
