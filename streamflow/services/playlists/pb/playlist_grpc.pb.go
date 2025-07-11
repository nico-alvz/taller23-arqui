// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// PlaylistsServiceClient is the client API for PlaylistsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PlaylistsServiceClient interface {
	CreatePlaylist(ctx context.Context, in *CreatePlaylistRequest, opts ...grpc.CallOption) (*CreatePlaylistResponse, error)
	AddVideo(ctx context.Context, in *AddVideoRequest, opts ...grpc.CallOption) (*AddVideoResponse, error)
	RemoveVideo(ctx context.Context, in *RemoveVideoRequest, opts ...grpc.CallOption) (*RemoveVideoResponse, error)
	ListPlaylists(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListPlaylistsResponse, error)
	ListVideos(ctx context.Context, in *ListVideosRequest, opts ...grpc.CallOption) (*ListVideosResponse, error)
	DeletePlaylist(ctx context.Context, in *DeletePlaylistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type playlistsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPlaylistsServiceClient(cc grpc.ClientConnInterface) PlaylistsServiceClient {
	return &playlistsServiceClient{cc}
}

func (c *playlistsServiceClient) CreatePlaylist(ctx context.Context, in *CreatePlaylistRequest, opts ...grpc.CallOption) (*CreatePlaylistResponse, error) {
	out := new(CreatePlaylistResponse)
	err := c.cc.Invoke(ctx, "/PlaylistsService.PlaylistsService/CreatePlaylist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *playlistsServiceClient) AddVideo(ctx context.Context, in *AddVideoRequest, opts ...grpc.CallOption) (*AddVideoResponse, error) {
	out := new(AddVideoResponse)
	err := c.cc.Invoke(ctx, "/PlaylistsService.PlaylistsService/AddVideo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *playlistsServiceClient) RemoveVideo(ctx context.Context, in *RemoveVideoRequest, opts ...grpc.CallOption) (*RemoveVideoResponse, error) {
	out := new(RemoveVideoResponse)
	err := c.cc.Invoke(ctx, "/PlaylistsService.PlaylistsService/RemoveVideo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *playlistsServiceClient) ListPlaylists(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListPlaylistsResponse, error) {
	out := new(ListPlaylistsResponse)
	err := c.cc.Invoke(ctx, "/PlaylistsService.PlaylistsService/ListPlaylists", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *playlistsServiceClient) ListVideos(ctx context.Context, in *ListVideosRequest, opts ...grpc.CallOption) (*ListVideosResponse, error) {
	out := new(ListVideosResponse)
	err := c.cc.Invoke(ctx, "/PlaylistsService.PlaylistsService/ListVideos", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *playlistsServiceClient) DeletePlaylist(ctx context.Context, in *DeletePlaylistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/PlaylistsService.PlaylistsService/DeletePlaylist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PlaylistsServiceServer is the server API for PlaylistsService service.
// All implementations must embed UnimplementedPlaylistsServiceServer
// for forward compatibility
type PlaylistsServiceServer interface {
	CreatePlaylist(context.Context, *CreatePlaylistRequest) (*CreatePlaylistResponse, error)
	AddVideo(context.Context, *AddVideoRequest) (*AddVideoResponse, error)
	RemoveVideo(context.Context, *RemoveVideoRequest) (*RemoveVideoResponse, error)
	ListPlaylists(context.Context, *emptypb.Empty) (*ListPlaylistsResponse, error)
	ListVideos(context.Context, *ListVideosRequest) (*ListVideosResponse, error)
	DeletePlaylist(context.Context, *DeletePlaylistRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedPlaylistsServiceServer()
}

// UnimplementedPlaylistsServiceServer must be embedded to have forward compatible implementations.
type UnimplementedPlaylistsServiceServer struct {
}

func (UnimplementedPlaylistsServiceServer) CreatePlaylist(context.Context, *CreatePlaylistRequest) (*CreatePlaylistResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreatePlaylist not implemented")
}
func (UnimplementedPlaylistsServiceServer) AddVideo(context.Context, *AddVideoRequest) (*AddVideoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddVideo not implemented")
}
func (UnimplementedPlaylistsServiceServer) RemoveVideo(context.Context, *RemoveVideoRequest) (*RemoveVideoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveVideo not implemented")
}
func (UnimplementedPlaylistsServiceServer) ListPlaylists(context.Context, *emptypb.Empty) (*ListPlaylistsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPlaylists not implemented")
}
func (UnimplementedPlaylistsServiceServer) ListVideos(context.Context, *ListVideosRequest) (*ListVideosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListVideos not implemented")
}
func (UnimplementedPlaylistsServiceServer) DeletePlaylist(context.Context, *DeletePlaylistRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeletePlaylist not implemented")
}
func (UnimplementedPlaylistsServiceServer) mustEmbedUnimplementedPlaylistsServiceServer() {}

// UnsafePlaylistsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PlaylistsServiceServer will
// result in compilation errors.
type UnsafePlaylistsServiceServer interface {
	mustEmbedUnimplementedPlaylistsServiceServer()
}

func RegisterPlaylistsServiceServer(s *grpc.Server, srv PlaylistsServiceServer) {
	s.RegisterService(&_PlaylistsService_serviceDesc, srv)
}

func _PlaylistsService_CreatePlaylist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreatePlaylistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlaylistsServiceServer).CreatePlaylist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/PlaylistsService.PlaylistsService/CreatePlaylist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlaylistsServiceServer).CreatePlaylist(ctx, req.(*CreatePlaylistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PlaylistsService_AddVideo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddVideoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlaylistsServiceServer).AddVideo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/PlaylistsService.PlaylistsService/AddVideo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlaylistsServiceServer).AddVideo(ctx, req.(*AddVideoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PlaylistsService_RemoveVideo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveVideoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlaylistsServiceServer).RemoveVideo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/PlaylistsService.PlaylistsService/RemoveVideo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlaylistsServiceServer).RemoveVideo(ctx, req.(*RemoveVideoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PlaylistsService_ListPlaylists_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlaylistsServiceServer).ListPlaylists(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/PlaylistsService.PlaylistsService/ListPlaylists",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlaylistsServiceServer).ListPlaylists(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _PlaylistsService_ListVideos_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListVideosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlaylistsServiceServer).ListVideos(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/PlaylistsService.PlaylistsService/ListVideos",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlaylistsServiceServer).ListVideos(ctx, req.(*ListVideosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PlaylistsService_DeletePlaylist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeletePlaylistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlaylistsServiceServer).DeletePlaylist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/PlaylistsService.PlaylistsService/DeletePlaylist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlaylistsServiceServer).DeletePlaylist(ctx, req.(*DeletePlaylistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _PlaylistsService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "PlaylistsService.PlaylistsService",
	HandlerType: (*PlaylistsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreatePlaylist",
			Handler:    _PlaylistsService_CreatePlaylist_Handler,
		},
		{
			MethodName: "AddVideo",
			Handler:    _PlaylistsService_AddVideo_Handler,
		},
		{
			MethodName: "RemoveVideo",
			Handler:    _PlaylistsService_RemoveVideo_Handler,
		},
		{
			MethodName: "ListPlaylists",
			Handler:    _PlaylistsService_ListPlaylists_Handler,
		},
		{
			MethodName: "ListVideos",
			Handler:    _PlaylistsService_ListVideos_Handler,
		},
		{
			MethodName: "DeletePlaylist",
			Handler:    _PlaylistsService_DeletePlaylist_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "playlist.proto",
}
