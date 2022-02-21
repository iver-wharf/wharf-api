// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package v5

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BuildsClient is the client API for Builds service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BuildsClient interface {
	// CreateLogStream allows creating logs as a client-side stream.
	// Logs targeting non-existing builds as well as logs that has already been
	// added before (based on the build, log, and step IDs) will be discarded.
	CreateLogStream(ctx context.Context, opts ...grpc.CallOption) (Builds_CreateLogStreamClient, error)
}

type buildsClient struct {
	cc grpc.ClientConnInterface
}

func NewBuildsClient(cc grpc.ClientConnInterface) BuildsClient {
	return &buildsClient{cc}
}

func (c *buildsClient) CreateLogStream(ctx context.Context, opts ...grpc.CallOption) (Builds_CreateLogStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &Builds_ServiceDesc.Streams[0], "/wharf.api.v5.Builds/CreateLogStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &buildsCreateLogStreamClient{stream}
	return x, nil
}

type Builds_CreateLogStreamClient interface {
	Send(*CreateLogStreamRequest) error
	CloseAndRecv() (*CreateLogStreamResponse, error)
	grpc.ClientStream
}

type buildsCreateLogStreamClient struct {
	grpc.ClientStream
}

func (x *buildsCreateLogStreamClient) Send(m *CreateLogStreamRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *buildsCreateLogStreamClient) CloseAndRecv() (*CreateLogStreamResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(CreateLogStreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// BuildsServer is the server API for Builds service.
// All implementations must embed UnimplementedBuildsServer
// for forward compatibility
type BuildsServer interface {
	// CreateLogStream allows creating logs as a client-side stream.
	// Logs targeting non-existing builds as well as logs that has already been
	// added before (based on the build, log, and step IDs) will be discarded.
	CreateLogStream(Builds_CreateLogStreamServer) error
	mustEmbedUnimplementedBuildsServer()
}

// UnimplementedBuildsServer must be embedded to have forward compatible implementations.
type UnimplementedBuildsServer struct {
}

func (UnimplementedBuildsServer) CreateLogStream(Builds_CreateLogStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method CreateLogStream not implemented")
}
func (UnimplementedBuildsServer) mustEmbedUnimplementedBuildsServer() {}

// UnsafeBuildsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BuildsServer will
// result in compilation errors.
type UnsafeBuildsServer interface {
	mustEmbedUnimplementedBuildsServer()
}

func RegisterBuildsServer(s grpc.ServiceRegistrar, srv BuildsServer) {
	s.RegisterService(&Builds_ServiceDesc, srv)
}

func _Builds_CreateLogStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BuildsServer).CreateLogStream(&buildsCreateLogStreamServer{stream})
}

type Builds_CreateLogStreamServer interface {
	SendAndClose(*CreateLogStreamResponse) error
	Recv() (*CreateLogStreamRequest, error)
	grpc.ServerStream
}

type buildsCreateLogStreamServer struct {
	grpc.ServerStream
}

func (x *buildsCreateLogStreamServer) SendAndClose(m *CreateLogStreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *buildsCreateLogStreamServer) Recv() (*CreateLogStreamRequest, error) {
	m := new(CreateLogStreamRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Builds_ServiceDesc is the grpc.ServiceDesc for Builds service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Builds_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "wharf.api.v5.Builds",
	HandlerType: (*BuildsServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "CreateLogStream",
			Handler:       _Builds_CreateLogStream_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "api/wharfapi/v5/builds.proto",
}
