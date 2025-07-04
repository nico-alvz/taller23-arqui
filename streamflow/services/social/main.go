package main

import (
    "context"
    "log"
    "net"
    "sync"
     
    "github.com/google/uuid"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"

  	//amqp "github.com/rabbitmq/amqp091-go"

    
    pb "social-service/pb" // Adjust the import path as needed
)


// ================================
//  In-memory store (thread-safe)
// ================================

type store struct {
    mu       sync.RWMutex
    likes    map[string][]*pb.Like    // key: video_id
    comments map[string][]*pb.Comment // key: video_id
}

func newStore() *store {
    return &store{
        likes:    make(map[string][]*pb.Like),
        comments: make(map[string][]*pb.Comment),
    }
}

// ================================
//  gRPC server implementation
// ================================

type server struct {
    pb.UnimplementedSocialInteractionsServer
    db *store
}

func (s *server) LikeVideo(ctx context.Context, req *pb.LikeVideoRequest) (*pb.LikeVideoResponse, error) {
    userID := req.GetUserId()
    if userID == "" {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }

    like := &pb.Like{
        LikeId:   uuid.New().String(),
        UserId:   userID,
        CreatedAt: timestamppb.Now(),
    }

    s.db.mu.Lock()
    s.db.likes[req.GetVideoId()] = append(s.db.likes[req.GetVideoId()], like)
    s.db.mu.Unlock()

    return &pb.LikeVideoResponse{Like: like}, nil
}

func (s *server) CommentVideo(ctx context.Context, req *pb.CommentVideoRequest) (*pb.CommentVideoResponse, error) {
    userID := req.GetUserId()
    if userID == "" {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }

    comment := &pb.Comment{
        CommentId: uuid.New().String(),
        UserId:    userID,
        Content:   req.GetContent(),
        CreatedAt: timestamppb.Now(),
    }

    s.db.mu.Lock()
    s.db.comments[req.GetVideoId()] = append(s.db.comments[req.GetVideoId()], comment)
    s.db.mu.Unlock()

    return &pb.CommentVideoResponse{Comment: comment}, nil
}

func (s *server) GetVideoInteractions(ctx context.Context, req *pb.GetVideoInteractionsRequest) (*pb.GetVideoInteractionsResponse, error) {
    userID := req.GetUserId()
    if userID == "" {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }

    s.db.mu.RLock()
    likes := s.db.likes[req.GetVideoId()]
    comments := s.db.comments[req.GetVideoId()]
    s.db.mu.RUnlock()

    if likes == nil {
        likes = []*pb.Like{}
    }
    if comments == nil {
        comments = []*pb.Comment{}
    }

    return &pb.GetVideoInteractionsResponse{
        Likes:    likes,
        Comments: comments,
    }, nil
}

// ================================
//  Authentication Interceptor
// ================================

func authUnaryInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }
    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing auth token")
    }
    ctx = context.WithValue(ctx, userIDKey{}, tokens[0])
    return handler(ctx, req)
}

type userIDKey struct{}

func userIDFromContext(ctx context.Context) (string, bool) {
    v := ctx.Value(userIDKey{})
    if v == nil {
        return "", false
    }
    return v.(string), true
}

// ================================
//  main
// ================================

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(authUnaryInterceptor),
    )

    srv := &server{db: newStore()}
    pb.RegisterSocialInteractionsServer(grpcServer, srv)

    log.Println("gRPC server listening on :50051")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
