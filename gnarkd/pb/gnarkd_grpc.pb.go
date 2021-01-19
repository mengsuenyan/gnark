// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pb

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

// Groth16Client is the client API for Groth16 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type Groth16Client interface {
	// Prove takes circuitID and optional witness as parameter. If optional witness is not specified
	// ProveJobStatus will be in a status "awaiting for witness" which must be sent outside gRPC
	// through a TCP connection. This ensure that the API can deal with large witnesses.
	// For small circuits, ProveResult may contain the proof. For large circuits, must use JobStatus and
	// await for async result
	Prove(ctx context.Context, in *ProveRequest, opts ...grpc.CallOption) (*ProveResult, error)
	// JobStatus is a bidirectional stream enabling clients to regularly poll the server to get their job status
	JobStatus(ctx context.Context, opts ...grpc.CallOption) (Groth16_JobStatusClient, error)
}

type groth16Client struct {
	cc grpc.ClientConnInterface
}

func NewGroth16Client(cc grpc.ClientConnInterface) Groth16Client {
	return &groth16Client{cc}
}

func (c *groth16Client) Prove(ctx context.Context, in *ProveRequest, opts ...grpc.CallOption) (*ProveResult, error) {
	out := new(ProveResult)
	err := c.cc.Invoke(ctx, "/gnarkd.Groth16/Prove", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *groth16Client) JobStatus(ctx context.Context, opts ...grpc.CallOption) (Groth16_JobStatusClient, error) {
	stream, err := c.cc.NewStream(ctx, &Groth16_ServiceDesc.Streams[0], "/gnarkd.Groth16/JobStatus", opts...)
	if err != nil {
		return nil, err
	}
	x := &groth16JobStatusClient{stream}
	return x, nil
}

type Groth16_JobStatusClient interface {
	Send(*JobStatusRequest) error
	Recv() (*ProveResult, error)
	grpc.ClientStream
}

type groth16JobStatusClient struct {
	grpc.ClientStream
}

func (x *groth16JobStatusClient) Send(m *JobStatusRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *groth16JobStatusClient) Recv() (*ProveResult, error) {
	m := new(ProveResult)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Groth16Server is the server API for Groth16 service.
// All implementations must embed UnimplementedGroth16Server
// for forward compatibility
type Groth16Server interface {
	// Prove takes circuitID and optional witness as parameter. If optional witness is not specified
	// ProveJobStatus will be in a status "awaiting for witness" which must be sent outside gRPC
	// through a TCP connection. This ensure that the API can deal with large witnesses.
	// For small circuits, ProveResult may contain the proof. For large circuits, must use JobStatus and
	// await for async result
	Prove(context.Context, *ProveRequest) (*ProveResult, error)
	// JobStatus is a bidirectional stream enabling clients to regularly poll the server to get their job status
	JobStatus(Groth16_JobStatusServer) error
	mustEmbedUnimplementedGroth16Server()
}

// UnimplementedGroth16Server must be embedded to have forward compatible implementations.
type UnimplementedGroth16Server struct {
}

func (UnimplementedGroth16Server) Prove(context.Context, *ProveRequest) (*ProveResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Prove not implemented")
}
func (UnimplementedGroth16Server) JobStatus(Groth16_JobStatusServer) error {
	return status.Errorf(codes.Unimplemented, "method JobStatus not implemented")
}
func (UnimplementedGroth16Server) mustEmbedUnimplementedGroth16Server() {}

// UnsafeGroth16Server may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to Groth16Server will
// result in compilation errors.
type UnsafeGroth16Server interface {
	mustEmbedUnimplementedGroth16Server()
}

func RegisterGroth16Server(s grpc.ServiceRegistrar, srv Groth16Server) {
	s.RegisterService(&Groth16_ServiceDesc, srv)
}

func _Groth16_Prove_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Groth16Server).Prove(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gnarkd.Groth16/Prove",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Groth16Server).Prove(ctx, req.(*ProveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Groth16_JobStatus_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(Groth16Server).JobStatus(&groth16JobStatusServer{stream})
}

type Groth16_JobStatusServer interface {
	Send(*ProveResult) error
	Recv() (*JobStatusRequest, error)
	grpc.ServerStream
}

type groth16JobStatusServer struct {
	grpc.ServerStream
}

func (x *groth16JobStatusServer) Send(m *ProveResult) error {
	return x.ServerStream.SendMsg(m)
}

func (x *groth16JobStatusServer) Recv() (*JobStatusRequest, error) {
	m := new(JobStatusRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Groth16_ServiceDesc is the grpc.ServiceDesc for Groth16 service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Groth16_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gnarkd.Groth16",
	HandlerType: (*Groth16Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Prove",
			Handler:    _Groth16_Prove_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "JobStatus",
			Handler:       _Groth16_JobStatus_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "pb/gnarkd.proto",
}
