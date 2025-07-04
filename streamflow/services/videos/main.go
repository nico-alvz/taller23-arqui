package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "videos-service/pb" // => generated files (users_pb2.go etc.)
)

// ----------- Config ---------------------------------------------------------

var (
	port       = getEnv("PORT", "50053")
	mongoURI   = getEnv("MONGODB_URI", "mongodb://localhost:27017/videos_db")
	dbName     = "videos_db"
	collection = "videos"
)

// ----------- Utils ----------------------------------------------------------

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ----------- Mongo model ----------------------------------------------------

type Video struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Title      string             `bson:"title"`
	Description string            `bson:"description"`
	Genre      string             `bson:"genre"`
	UploadDate time.Time          `bson:"upload_date"`
	DeletedAt  *time.Time         `bson:"deleted_at,omitempty"`
}

// Map Video ↔ proto.Video
func toProto(v *Video) *pb.Video {
	return &pb.Video{
		Id:          v.ID.Hex(),
		Title:       v.Title,
		Description: v.Description,
		Genre:       v.Genre,
		LikesCount:  0,
		UploadDate:  v.UploadDate.Format(time.RFC3339),
	}
}

func toVideoResponse(v *Video) *pb.VideoResponse {
	return &pb.VideoResponse{
		Id:          v.ID.Hex(),
		Title:       v.Title,
		Description: v.Description,
		Genre:       v.Genre,
		LikesCount:  0, // o el valor real si lo tienes
	}
}

// ----------- gRPC service ---------------------------------------------------

type videoServiceServer struct {
	pb.UnimplementedVideoServiceServer
	col *mongo.Collection
}

// UploadVideo
func (s *videoServiceServer) UploadVideo(ctx context.Context, req *pb.UploadVideoRequest) (*pb.VideoResponse, error) {
	v := Video{
		Title:       req.Title,
		Description: req.Description,
		Genre:       req.Genre,
		UploadDate:  time.Now(),
	}

	res, err := s.col.InsertOne(ctx, v)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DB insert error: %v", err)
	}
	v.ID = res.InsertedID.(primitive.ObjectID)
	log.Printf("Video creado: %s", v.Title)
	return toVideoResponse(&v), nil
}

// GetVideo
func (s *videoServiceServer) GetVideo(ctx context.Context, req *pb.GetVideoRequest) (*pb.VideoResponse, error) {
	objID, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ID inválido")
	}
	var v Video
	err = s.col.FindOne(ctx, bson.M{"_id": objID, "deleted_at": nil}).Decode(&v)
	if err == mongo.ErrNoDocuments {
		return nil, status.Errorf(codes.NotFound, "Video no encontrado")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DB error: %v", err)
	}
	return toVideoResponse(&v), nil
}

// UpdateVideo
func (s *videoServiceServer) UpdateVideo(ctx context.Context, req *pb.UpdateVideoRequest) (*pb.VideoResponse, error) {
	objID, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ID inválido")
	}
	update := bson.M{
		"$set": bson.M{
			"title":       req.Title,
			"description": req.Description,
			"genre":       req.Genre,
		},
	}
	var v Video
	err = s.col.FindOneAndUpdate(ctx,
		bson.M{"_id": objID, "deleted_at": nil},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&v)

	if err == mongo.ErrNoDocuments {
		return nil, status.Errorf(codes.NotFound, "Video no encontrado")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DB error: %v", err)
	}
	log.Printf("Video actualizado: %s", v.Title)
	return toVideoResponse(&v), nil
}

// DeleteVideo (soft delete)
func (s *videoServiceServer) DeleteVideo(ctx context.Context, req *pb.DeleteVideoRequest) (*pb.DeleteVideoResponse, error) {
	objID, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ID inválido")
	}
	now := time.Now()
	res, err := s.col.UpdateOne(ctx,
		bson.M{"_id": objID, "deleted_at": nil},
		bson.M{"$set": bson.M{"deleted_at": now}},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DB error: %v", err)
	}
	if res.MatchedCount == 0 {
		return nil, status.Errorf(codes.NotFound, "Video no encontrado")
	}
	log.Printf("Video eliminado: %s", req.Id)
	return &pb.DeleteVideoResponse{Message: "Video eliminado exitosamente"}, nil
}

// ListVideos
func (s *videoServiceServer) ListVideos(ctx context.Context, req *pb.ListVideosRequest) (*pb.ListVideosResponse, error) {
	filter := bson.M{"deleted_at": nil}
	if req.Title != "" {
		filter["title"] = bson.M{"$regex": req.Title, "$options": "i"}
	}
	if req.Genre != "" {
		filter["genre"] = bson.M{"$regex": req.Genre, "$options": "i"}
	}

	cur, err := s.col.Find(ctx, filter, options.Find().SetSort(bson.M{"upload_date": -1}))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DB error: %v", err)
	}
	defer cur.Close(ctx)

	var videos []*pb.Video
	for cur.Next(ctx) {
		var v Video
		if err := cur.Decode(&v); err != nil {
			return nil, status.Errorf(codes.Internal, "Decode error: %v", err)
		}
		videos = append(videos, toProto(&v))
	}
	return &pb.ListVideosResponse{Videos: videos}, nil
}

// ----------- Main -----------------------------------------------------------

func main() {
	// Mongo ⟶ colección
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Mongo connect error: %v", err)
	}
	col := client.Database(dbName).Collection(collection)

	// gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterVideoServiceServer(grpcServer, &videoServiceServer{col: col})

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// HTTP health-check
	httpPort, _ := strconv.Atoi(port)
	go func() {
		hPort := httpPort + 1000
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"healthy","service":"videos"}`)
		})
		log.Printf("HTTP health listening on %d", hPort)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", hPort), nil))
	}()

	log.Printf("gRPC listening on %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}