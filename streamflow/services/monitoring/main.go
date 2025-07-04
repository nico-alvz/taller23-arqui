package main

import (
    "context"
    "log"
    "net"
    "time"
    "os"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/emptypb"
    "google.golang.org/protobuf/types/known/timestamppb"

    pb "monitoring-service/pb"
)

// MongoDB collections
var (
    mongoClient *mongo.Client
    actionsCollection *mongo.Collection
    errorsCollection *mongo.Collection
)

// Document structures
type ActionDoc struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    UserID    int64             `bson:"user_id,omitempty"`
    Email     string            `bson:"email,omitempty"`
    Method    string            `bson:"method"`
    URL       string            `bson:"url"`
    Action    string            `bson:"action"`
    CreatedAt time.Time         `bson:"created_at"`
}

type ErrorDoc struct {
    ID           primitive.ObjectID `bson:"_id,omitempty"`
    UserID       int64             `bson:"user_id,omitempty"`
    Email        string            `bson:"email,omitempty"`
    ErrorMessage string            `bson:"error_message"`
    CreatedAt    time.Time         `bson:"created_at"`
}

// Utility functions
func getenv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

// Initialize MongoDB connection
func initMongoDB() error {
    mongoURI := getenv("MONGODB_URI", "mongodb://root:password@localhost:27017/monitoring_db?authSource=admin")
    
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
    if err != nil {
        return err
    }
    
    // Test connection
    if err := client.Ping(context.Background(), nil); err != nil {
        return err
    }
    
    mongoClient = client
    db := client.Database("monitoring_db")
    actionsCollection = db.Collection("actions")
    errorsCollection = db.Collection("errors")
    
    log.Println("MongoDB monitoring database connected")
    return nil
}

// Monitoring service implementation
type monitoringServer struct {
    pb.UnimplementedMonitoringServiceServer
}

// ListActions returns all monitoring actions
func (s *monitoringServer) ListActions(ctx context.Context, _ *emptypb.Empty) (*pb.ListActionsResponse, error) {
    // Find all actions, sorted by creation time (newest first)
    cursor, err := actionsCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(100))
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to query actions: %v", err)
    }
    defer cursor.Close(ctx)
    
    var actions []*pb.ActionLog
    for cursor.Next(ctx) {
        var doc ActionDoc
        if err := cursor.Decode(&doc); err != nil {
            continue // Skip invalid documents
        }
        
        // Convert ObjectID to int64 (use timestamp portion)
        id := int64(doc.ID.Timestamp().Unix())
        
        actions = append(actions, &pb.ActionLog{
            Id:        id,
            Timestamp: timestamppb.New(doc.CreatedAt),
            UserId:    doc.UserID,
            Email:     doc.Email,
            Method:    doc.Method,
            Url:       doc.URL,
            Action:    doc.Action,
        })
    }
    
    return &pb.ListActionsResponse{Actions: actions}, nil
}

// ListErrors returns all monitoring errors
func (s *monitoringServer) ListErrors(ctx context.Context, _ *emptypb.Empty) (*pb.ListErrorsResponse, error) {
    // Find all errors, sorted by creation time (newest first)
    cursor, err := errorsCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(100))
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to query errors: %v", err)
    }
    defer cursor.Close(ctx)
    
    var errors []*pb.ErrorLog
    for cursor.Next(ctx) {
        var doc ErrorDoc
        if err := cursor.Decode(&doc); err != nil {
            continue // Skip invalid documents
        }
        
        // Convert ObjectID to int64 (use timestamp portion)
        id := int64(doc.ID.Timestamp().Unix())
        
        errors = append(errors, &pb.ErrorLog{
            Id:           id,
            Timestamp:    timestamppb.New(doc.CreatedAt),
            UserId:       doc.UserID,
            Email:        doc.Email,
            ErrorMessage: doc.ErrorMessage,
        })
    }
    
    return &pb.ListErrorsResponse{Errors: errors}, nil
}

// Log action for monitoring
func logAction(userID int64, email, method, url, action string) {
    doc := ActionDoc{
        UserID:    userID,
        Email:     email,
        Method:    method,
        URL:       url,
        Action:    action,
        CreatedAt: time.Now(),
    }
    
    _, err := actionsCollection.InsertOne(context.Background(), doc)
    if err != nil {
        log.Printf("Failed to log action: %v", err)
    }
}

// Log error for monitoring
func logError(userID int64, email, errorMessage string) {
    doc := ErrorDoc{
        UserID:       userID,
        Email:        email,
        ErrorMessage: errorMessage,
        CreatedAt:    time.Now(),
    }
    
    _, err := errorsCollection.InsertOne(context.Background(), doc)
    if err != nil {
        log.Printf("Failed to log error: %v", err)
    }
}

func main() {
    // Initialize MongoDB
    if err := initMongoDB(); err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    defer mongoClient.Disconnect(context.Background())
    
    // Insert some sample data
    logAction(1, "admin@streamflow.com", "GET", "/api/users", "listed_users")
    logAction(2, "user@streamflow.com", "POST", "/api/videos", "uploaded_video")
    logError(1, "admin@streamflow.com", "Failed to connect to external service")
    
    // Start gRPC server
    port := getenv("PORT", "50054")
    lis, err := net.Listen("tcp", ":"+port)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    grpcServer := grpc.NewServer()
    pb.RegisterMonitoringServiceServer(grpcServer, &monitoringServer{})
    
    log.Printf("gRPC MonitoringService listening on :%s", port)
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
