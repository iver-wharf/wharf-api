// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.19.1
// source: api/wharfapi/v5/builds.proto

package v5

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// CreateLogStreamRequest contains the streamed log lines that meant to be
// created.
type CreateLogStreamRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Lines is an array of log lines to allow chunking log line streams.
	Lines []*NewLogLine `protobuf:"bytes,1,rep,name=lines,proto3" json:"lines,omitempty"`
}

func (x *CreateLogStreamRequest) Reset() {
	*x = CreateLogStreamRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_wharfapi_v5_builds_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateLogStreamRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateLogStreamRequest) ProtoMessage() {}

func (x *CreateLogStreamRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_wharfapi_v5_builds_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateLogStreamRequest.ProtoReflect.Descriptor instead.
func (*CreateLogStreamRequest) Descriptor() ([]byte, []int) {
	return file_api_wharfapi_v5_builds_proto_rawDescGZIP(), []int{0}
}

func (x *CreateLogStreamRequest) GetLines() []*NewLogLine {
	if x != nil {
		return x.Lines
	}
	return nil
}

// CreateLogStreamResponse is the response returned after closing a log creation
// stream.
type CreateLogStreamResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// LinesInserted is the number of lines that has been inserted in total by
	// this stream.
	LinesInserted uint64 `protobuf:"varint,1,opt,name=lines_inserted,json=linesInserted,proto3" json:"lines_inserted,omitempty"`
}

func (x *CreateLogStreamResponse) Reset() {
	*x = CreateLogStreamResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_wharfapi_v5_builds_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateLogStreamResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateLogStreamResponse) ProtoMessage() {}

func (x *CreateLogStreamResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_wharfapi_v5_builds_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateLogStreamResponse.ProtoReflect.Descriptor instead.
func (*CreateLogStreamResponse) Descriptor() ([]byte, []int) {
	return file_api_wharfapi_v5_builds_proto_rawDescGZIP(), []int{1}
}

func (x *CreateLogStreamResponse) GetLinesInserted() uint64 {
	if x != nil {
		return x.LinesInserted
	}
	return 0
}

// NewLogLine is a single log line produced by a Wharf build to be added to the
// database.
type NewLogLine struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// BuildId is the database ID of the build this log line belongs to.
	BuildId uint64 `protobuf:"varint,1,opt,name=build_id,json=buildId,proto3" json:"build_id,omitempty"`
	// LogId is the worker's own ID of the log line. It's unique per build step
	// for a given build, but may have collisons across multiple steps or builds.
	LogId uint64 `protobuf:"varint,2,opt,name=log_id,json=logId,proto3" json:"log_id,omitempty"` // not DB value; worker's own ID of the log line
	// StepId is the worker's own ID of the step. It's unique for a given build,
	// but may have collisons across multiple builds.
	StepId uint64 `protobuf:"varint,3,opt,name=step_id,json=stepId,proto3" json:"step_id,omitempty"` // not DB value; worker's own ID of the step ID
	// Timestamp is when the log line was outputted from the build step.
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// Message is the log line text.
	Message string `protobuf:"bytes,5,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *NewLogLine) Reset() {
	*x = NewLogLine{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_wharfapi_v5_builds_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NewLogLine) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewLogLine) ProtoMessage() {}

func (x *NewLogLine) ProtoReflect() protoreflect.Message {
	mi := &file_api_wharfapi_v5_builds_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewLogLine.ProtoReflect.Descriptor instead.
func (*NewLogLine) Descriptor() ([]byte, []int) {
	return file_api_wharfapi_v5_builds_proto_rawDescGZIP(), []int{2}
}

func (x *NewLogLine) GetBuildId() uint64 {
	if x != nil {
		return x.BuildId
	}
	return 0
}

func (x *NewLogLine) GetLogId() uint64 {
	if x != nil {
		return x.LogId
	}
	return 0
}

func (x *NewLogLine) GetStepId() uint64 {
	if x != nil {
		return x.StepId
	}
	return 0
}

func (x *NewLogLine) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *NewLogLine) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_api_wharfapi_v5_builds_proto protoreflect.FileDescriptor

var file_api_wharfapi_v5_builds_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x61, 0x70, 0x69, 0x2f, 0x77, 0x68, 0x61, 0x72, 0x66, 0x61, 0x70, 0x69, 0x2f, 0x76,
	0x35, 0x2f, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c,
	0x77, 0x68, 0x61, 0x72, 0x66, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x35, 0x1a, 0x1f, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x48, 0x0a,
	0x16, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2e, 0x0a, 0x05, 0x6c, 0x69, 0x6e, 0x65, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x77, 0x68, 0x61, 0x72, 0x66, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x76, 0x35, 0x2e, 0x4e, 0x65, 0x77, 0x4c, 0x6f, 0x67, 0x4c, 0x69, 0x6e, 0x65,
	0x52, 0x05, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x22, 0x40, 0x0a, 0x17, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x4c, 0x6f, 0x67, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x25, 0x0a, 0x0e, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x5f, 0x69, 0x6e, 0x73, 0x65,
	0x72, 0x74, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0d, 0x6c, 0x69, 0x6e, 0x65,
	0x73, 0x49, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x65, 0x64, 0x22, 0xab, 0x01, 0x0a, 0x0a, 0x4e, 0x65,
	0x77, 0x4c, 0x6f, 0x67, 0x4c, 0x69, 0x6e, 0x65, 0x12, 0x19, 0x0a, 0x08, 0x62, 0x75, 0x69, 0x6c,
	0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x62, 0x75, 0x69, 0x6c,
	0x64, 0x49, 0x64, 0x12, 0x15, 0x0a, 0x06, 0x6c, 0x6f, 0x67, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x05, 0x6c, 0x6f, 0x67, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x73, 0x74,
	0x65, 0x70, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x73, 0x74, 0x65,
	0x70, 0x49, 0x64, 0x12, 0x38, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x18, 0x0a,
	0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0x6a, 0x0a, 0x06, 0x42, 0x75, 0x69, 0x6c, 0x64,
	0x73, 0x12, 0x60, 0x0a, 0x0f, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x53, 0x74,
	0x72, 0x65, 0x61, 0x6d, 0x12, 0x24, 0x2e, 0x77, 0x68, 0x61, 0x72, 0x66, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x76, 0x35, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x53, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x25, 0x2e, 0x77, 0x68, 0x61,
	0x72, 0x66, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x35, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x4c, 0x6f, 0x67, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x28, 0x01, 0x42, 0x34, 0x5a, 0x32, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x69, 0x76, 0x65, 0x72, 0x2d, 0x77, 0x68, 0x61, 0x72, 0x66, 0x2f, 0x77, 0x68, 0x61,
	0x72, 0x66, 0x2d, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x35, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x77, 0x68,
	0x61, 0x72, 0x66, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x35, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_api_wharfapi_v5_builds_proto_rawDescOnce sync.Once
	file_api_wharfapi_v5_builds_proto_rawDescData = file_api_wharfapi_v5_builds_proto_rawDesc
)

func file_api_wharfapi_v5_builds_proto_rawDescGZIP() []byte {
	file_api_wharfapi_v5_builds_proto_rawDescOnce.Do(func() {
		file_api_wharfapi_v5_builds_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_wharfapi_v5_builds_proto_rawDescData)
	})
	return file_api_wharfapi_v5_builds_proto_rawDescData
}

var file_api_wharfapi_v5_builds_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_wharfapi_v5_builds_proto_goTypes = []interface{}{
	(*CreateLogStreamRequest)(nil),  // 0: wharf.api.v5.CreateLogStreamRequest
	(*CreateLogStreamResponse)(nil), // 1: wharf.api.v5.CreateLogStreamResponse
	(*NewLogLine)(nil),              // 2: wharf.api.v5.NewLogLine
	(*timestamppb.Timestamp)(nil),   // 3: google.protobuf.Timestamp
}
var file_api_wharfapi_v5_builds_proto_depIdxs = []int32{
	2, // 0: wharf.api.v5.CreateLogStreamRequest.lines:type_name -> wharf.api.v5.NewLogLine
	3, // 1: wharf.api.v5.NewLogLine.timestamp:type_name -> google.protobuf.Timestamp
	0, // 2: wharf.api.v5.Builds.CreateLogStream:input_type -> wharf.api.v5.CreateLogStreamRequest
	1, // 3: wharf.api.v5.Builds.CreateLogStream:output_type -> wharf.api.v5.CreateLogStreamResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_api_wharfapi_v5_builds_proto_init() }
func file_api_wharfapi_v5_builds_proto_init() {
	if File_api_wharfapi_v5_builds_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_wharfapi_v5_builds_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateLogStreamRequest); i {
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
		file_api_wharfapi_v5_builds_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateLogStreamResponse); i {
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
		file_api_wharfapi_v5_builds_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NewLogLine); i {
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
			RawDescriptor: file_api_wharfapi_v5_builds_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_wharfapi_v5_builds_proto_goTypes,
		DependencyIndexes: file_api_wharfapi_v5_builds_proto_depIdxs,
		MessageInfos:      file_api_wharfapi_v5_builds_proto_msgTypes,
	}.Build()
	File_api_wharfapi_v5_builds_proto = out.File
	file_api_wharfapi_v5_builds_proto_rawDesc = nil
	file_api_wharfapi_v5_builds_proto_goTypes = nil
	file_api_wharfapi_v5_builds_proto_depIdxs = nil
}
