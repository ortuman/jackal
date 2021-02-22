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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.6.1
// source: jackal/proto/authenticator/v1/authenticator.proto

package v1

import (
	context "context"
	reflect "reflect"
	sync "sync"

	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

// AuthenticateRequest is the parameter message for Authenticate rpc.
type AuthenticateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// username is the name of the user who's trying to authenticates.
	Username string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	// password is the user input password.
	Password string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *AuthenticateRequest) Reset() {
	*x = AuthenticateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AuthenticateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthenticateRequest) ProtoMessage() {}

func (x *AuthenticateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthenticateRequest.ProtoReflect.Descriptor instead.
func (*AuthenticateRequest) Descriptor() ([]byte, []int) {
	return file_jackal_proto_authenticator_v1_authenticator_proto_rawDescGZIP(), []int{0}
}

func (x *AuthenticateRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *AuthenticateRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

// AuthenticateResponse is the response returned by Authenticate rpc.
type AuthenticateResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// authenticated tells whether or not Authenticate operation was successful.
	Authenticated bool `protobuf:"varint,1,opt,name=authenticated,proto3" json:"authenticated,omitempty"`
}

func (x *AuthenticateResponse) Reset() {
	*x = AuthenticateResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AuthenticateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthenticateResponse) ProtoMessage() {}

func (x *AuthenticateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthenticateResponse.ProtoReflect.Descriptor instead.
func (*AuthenticateResponse) Descriptor() ([]byte, []int) {
	return file_jackal_proto_authenticator_v1_authenticator_proto_rawDescGZIP(), []int{1}
}

func (x *AuthenticateResponse) GetAuthenticated() bool {
	if x != nil {
		return x.Authenticated
	}
	return false
}

var File_jackal_proto_authenticator_v1_authenticator_proto protoreflect.FileDescriptor

var file_jackal_proto_authenticator_v1_authenticator_proto_rawDesc = []byte{
	0x0a, 0x31, 0x6a, 0x61, 0x63, 0x6b, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x61,
	0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x2f, 0x76, 0x31, 0x2f,
	0x61, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x6a, 0x61, 0x63, 0x6b, 0x61, 0x6c, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x61, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x76, 0x31,
	0x22, 0x4d, 0x0a, 0x13, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22,
	0x3c, 0x0a, 0x14, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x61, 0x75, 0x74, 0x68, 0x65,
	0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0d,
	0x61, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x64, 0x32, 0x84, 0x01,
	0x0a, 0x0d, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x12,
	0x73, 0x0a, 0x0c, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x12,
	0x30, 0x2e, 0x6a, 0x61, 0x63, 0x6b, 0x61, 0x6c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x61, 0x75, 0x74,
	0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x75,
	0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x31, 0x2e, 0x6a, 0x61, 0x63, 0x6b, 0x61, 0x6c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x61,
	0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x42, 0x72, 0x0a, 0x21, 0x6f, 0x72, 0x67, 0x2e, 0x6a, 0x61, 0x63, 0x6b,
	0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74,
	0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x76, 0x31, 0x42, 0x14, 0x41, 0x75, 0x74, 0x68, 0x65,
	0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x47, 0x72, 0x70, 0x63, 0x41, 0x50, 0x49, 0x50,
	0x01, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x61,
	0x63, 0x6b, 0x61, 0x6c, 0x2d, 0x78, 0x6d, 0x70, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
	0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x67, 0x6f, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69,
	0x63, 0x61, 0x74, 0x6f, 0x72, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_jackal_proto_authenticator_v1_authenticator_proto_rawDescOnce sync.Once
	file_jackal_proto_authenticator_v1_authenticator_proto_rawDescData = file_jackal_proto_authenticator_v1_authenticator_proto_rawDesc
)

func file_jackal_proto_authenticator_v1_authenticator_proto_rawDescGZIP() []byte {
	file_jackal_proto_authenticator_v1_authenticator_proto_rawDescOnce.Do(func() {
		file_jackal_proto_authenticator_v1_authenticator_proto_rawDescData = protoimpl.X.CompressGZIP(file_jackal_proto_authenticator_v1_authenticator_proto_rawDescData)
	})
	return file_jackal_proto_authenticator_v1_authenticator_proto_rawDescData
}

var file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_jackal_proto_authenticator_v1_authenticator_proto_goTypes = []interface{}{
	(*AuthenticateRequest)(nil),  // 0: jackal.api.authenticator.v1.AuthenticateRequest
	(*AuthenticateResponse)(nil), // 1: jackal.api.authenticator.v1.AuthenticateResponse
}
var file_jackal_proto_authenticator_v1_authenticator_proto_depIdxs = []int32{
	0, // 0: jackal.api.authenticator.v1.Authenticator.Authenticate:input_type -> jackal.api.authenticator.v1.AuthenticateRequest
	1, // 1: jackal.api.authenticator.v1.Authenticator.Authenticate:output_type -> jackal.api.authenticator.v1.AuthenticateResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_jackal_proto_authenticator_v1_authenticator_proto_init() }
func file_jackal_proto_authenticator_v1_authenticator_proto_init() {
	if File_jackal_proto_authenticator_v1_authenticator_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AuthenticateRequest); i {
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
		file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AuthenticateResponse); i {
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
			RawDescriptor: file_jackal_proto_authenticator_v1_authenticator_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_jackal_proto_authenticator_v1_authenticator_proto_goTypes,
		DependencyIndexes: file_jackal_proto_authenticator_v1_authenticator_proto_depIdxs,
		MessageInfos:      file_jackal_proto_authenticator_v1_authenticator_proto_msgTypes,
	}.Build()
	File_jackal_proto_authenticator_v1_authenticator_proto = out.File
	file_jackal_proto_authenticator_v1_authenticator_proto_rawDesc = nil
	file_jackal_proto_authenticator_v1_authenticator_proto_goTypes = nil
	file_jackal_proto_authenticator_v1_authenticator_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// AuthenticatorClient is the client API for Authenticator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AuthenticatorClient interface {
	// Authenticate method performs authentication given a username and password.
	Authenticate(ctx context.Context, in *AuthenticateRequest, opts ...grpc.CallOption) (*AuthenticateResponse, error)
}

type authenticatorClient struct {
	cc *grpc.ClientConn
}

func NewAuthenticatorClient(cc *grpc.ClientConn) AuthenticatorClient {
	return &authenticatorClient{cc}
}

func (c *authenticatorClient) Authenticate(ctx context.Context, in *AuthenticateRequest, opts ...grpc.CallOption) (*AuthenticateResponse, error) {
	out := new(AuthenticateResponse)
	err := c.cc.Invoke(ctx, "/jackal.api.authenticator.v1.Authenticator/Authenticate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticatorServer is the server API for Authenticator service.
type AuthenticatorServer interface {
	// Authenticate method performs authentication given a username and password.
	Authenticate(context.Context, *AuthenticateRequest) (*AuthenticateResponse, error)
}

// UnimplementedAuthenticatorServer can be embedded to have forward compatible implementations.
type UnimplementedAuthenticatorServer struct {
}

func (*UnimplementedAuthenticatorServer) Authenticate(context.Context, *AuthenticateRequest) (*AuthenticateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Authenticate not implemented")
}

func RegisterAuthenticatorServer(s *grpc.Server, srv AuthenticatorServer) {
	s.RegisterService(&_Authenticator_serviceDesc, srv)
}

func _Authenticator_Authenticate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthenticateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthenticatorServer).Authenticate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/jackal.api.authenticator.v1.Authenticator/Authenticate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthenticatorServer).Authenticate(ctx, req.(*AuthenticateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Authenticator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "jackal.api.authenticator.v1.Authenticator",
	HandlerType: (*AuthenticatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Authenticate",
			Handler:    _Authenticator_Authenticate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "jackal/proto/authenticator/v1/authenticator.proto",
}
