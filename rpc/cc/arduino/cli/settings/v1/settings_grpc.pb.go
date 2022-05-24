// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.17.3
// source: cc/arduino/cli/settings/v1/settings.proto

package settings

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

// SettingsServiceClient is the client API for SettingsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SettingsServiceClient interface {
	// List all the settings.
	GetAll(ctx context.Context, in *GetAllRequest, opts ...grpc.CallOption) (*GetAllResponse, error)
	// Set multiple settings values at once.
	Merge(ctx context.Context, in *MergeRequest, opts ...grpc.CallOption) (*MergeResponse, error)
	// Get the value of a specific setting.
	GetValue(ctx context.Context, in *GetValueRequest, opts ...grpc.CallOption) (*GetValueResponse, error)
	// Set the value of a specific setting.
	SetValue(ctx context.Context, in *SetValueRequest, opts ...grpc.CallOption) (*SetValueResponse, error)
	// Writes to file settings currently stored in memory
	Write(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error)
}

type settingsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSettingsServiceClient(cc grpc.ClientConnInterface) SettingsServiceClient {
	return &settingsServiceClient{cc}
}

func (c *settingsServiceClient) GetAll(ctx context.Context, in *GetAllRequest, opts ...grpc.CallOption) (*GetAllResponse, error) {
	out := new(GetAllResponse)
	err := c.cc.Invoke(ctx, "/cc.arduino.cli.settings.v1.SettingsService/GetAll", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *settingsServiceClient) Merge(ctx context.Context, in *MergeRequest, opts ...grpc.CallOption) (*MergeResponse, error) {
	out := new(MergeResponse)
	err := c.cc.Invoke(ctx, "/cc.arduino.cli.settings.v1.SettingsService/Merge", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *settingsServiceClient) GetValue(ctx context.Context, in *GetValueRequest, opts ...grpc.CallOption) (*GetValueResponse, error) {
	out := new(GetValueResponse)
	err := c.cc.Invoke(ctx, "/cc.arduino.cli.settings.v1.SettingsService/GetValue", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *settingsServiceClient) SetValue(ctx context.Context, in *SetValueRequest, opts ...grpc.CallOption) (*SetValueResponse, error) {
	out := new(SetValueResponse)
	err := c.cc.Invoke(ctx, "/cc.arduino.cli.settings.v1.SettingsService/SetValue", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *settingsServiceClient) Write(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := c.cc.Invoke(ctx, "/cc.arduino.cli.settings.v1.SettingsService/Write", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SettingsServiceServer is the server API for SettingsService service.
// All implementations must embed UnimplementedSettingsServiceServer
// for forward compatibility
type SettingsServiceServer interface {
	// List all the settings.
	GetAll(context.Context, *GetAllRequest) (*GetAllResponse, error)
	// Set multiple settings values at once.
	Merge(context.Context, *MergeRequest) (*MergeResponse, error)
	// Get the value of a specific setting.
	GetValue(context.Context, *GetValueRequest) (*GetValueResponse, error)
	// Set the value of a specific setting.
	SetValue(context.Context, *SetValueRequest) (*SetValueResponse, error)
	// Writes to file settings currently stored in memory
	Write(context.Context, *WriteRequest) (*WriteResponse, error)
	mustEmbedUnimplementedSettingsServiceServer()
}

// UnimplementedSettingsServiceServer must be embedded to have forward compatible implementations.
type UnimplementedSettingsServiceServer struct {
}

func (UnimplementedSettingsServiceServer) GetAll(context.Context, *GetAllRequest) (*GetAllResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAll not implemented")
}
func (UnimplementedSettingsServiceServer) Merge(context.Context, *MergeRequest) (*MergeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Merge not implemented")
}
func (UnimplementedSettingsServiceServer) GetValue(context.Context, *GetValueRequest) (*GetValueResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValue not implemented")
}
func (UnimplementedSettingsServiceServer) SetValue(context.Context, *SetValueRequest) (*SetValueResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetValue not implemented")
}
func (UnimplementedSettingsServiceServer) Write(context.Context, *WriteRequest) (*WriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Write not implemented")
}
func (UnimplementedSettingsServiceServer) mustEmbedUnimplementedSettingsServiceServer() {}

// UnsafeSettingsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SettingsServiceServer will
// result in compilation errors.
type UnsafeSettingsServiceServer interface {
	mustEmbedUnimplementedSettingsServiceServer()
}

func RegisterSettingsServiceServer(s grpc.ServiceRegistrar, srv SettingsServiceServer) {
	s.RegisterService(&SettingsService_ServiceDesc, srv)
}

func _SettingsService_GetAll_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAllRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SettingsServiceServer).GetAll(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cc.arduino.cli.settings.v1.SettingsService/GetAll",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SettingsServiceServer).GetAll(ctx, req.(*GetAllRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SettingsService_Merge_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MergeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SettingsServiceServer).Merge(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cc.arduino.cli.settings.v1.SettingsService/Merge",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SettingsServiceServer).Merge(ctx, req.(*MergeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SettingsService_GetValue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetValueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SettingsServiceServer).GetValue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cc.arduino.cli.settings.v1.SettingsService/GetValue",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SettingsServiceServer).GetValue(ctx, req.(*GetValueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SettingsService_SetValue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetValueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SettingsServiceServer).SetValue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cc.arduino.cli.settings.v1.SettingsService/SetValue",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SettingsServiceServer).SetValue(ctx, req.(*SetValueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SettingsService_Write_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SettingsServiceServer).Write(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cc.arduino.cli.settings.v1.SettingsService/Write",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SettingsServiceServer).Write(ctx, req.(*WriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// SettingsService_ServiceDesc is the grpc.ServiceDesc for SettingsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SettingsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cc.arduino.cli.settings.v1.SettingsService",
	HandlerType: (*SettingsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetAll",
			Handler:    _SettingsService_GetAll_Handler,
		},
		{
			MethodName: "Merge",
			Handler:    _SettingsService_Merge_Handler,
		},
		{
			MethodName: "GetValue",
			Handler:    _SettingsService_GetValue_Handler,
		},
		{
			MethodName: "SetValue",
			Handler:    _SettingsService_SetValue_Handler,
		},
		{
			MethodName: "Write",
			Handler:    _SettingsService_Write_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cc/arduino/cli/settings/v1/settings.proto",
}
