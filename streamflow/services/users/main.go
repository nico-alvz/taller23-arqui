package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

	_ "github.com/go-sql-driver/mysql"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "users-service/pb"
)

// Placeholder para protobuf generado
type server struct {
    pb.UnimplementedUserServiceServer
    db *sql.DB
    rabbitmq *amqp.Connection
    col *mongo.Collection
}

type User struct {
    ID        int       `json:"id"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Email     string    `json:"email"`
    Password  string    `json:"password"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
    DeletedAt *time.Time `json:"deleted_at"`
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func initDB() *sql.DB {
    dbHost := getEnv("DB_HOST", "localhost")
    dbPort := getEnv("DB_PORT", "3306")
    dbName := getEnv("DB_NAME", "users_db")
    dbUser := getEnv("DB_USER", "root")
    dbPassword := getEnv("DB_PASSWORD", "password")
    
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal("Error connecting to database:", err)
    }
    
    if err := db.Ping(); err != nil {
        log.Fatal("Error pinging database:", err)
    }
    
    // Crear tabla
    createTableQuery := `
    CREATE TABLE IF NOT EXISTS users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        first_name VARCHAR(50) NOT NULL,
        last_name VARCHAR(50) NOT NULL,
        email VARCHAR(100) NOT NULL UNIQUE,
        password VARCHAR(255) NOT NULL,
        role ENUM('Administrador', 'Cliente') NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMP NULL,
        INDEX idx_email (email),
        INDEX idx_deleted (deleted_at)
    )`
    
    if _, err := db.Exec(createTableQuery); err != nil {
        log.Fatal("Error creating users table:", err)
    }
    
    log.Println("Users database connected and initialized")
    return db
}

func initRabbitMQ() *amqp.Connection {
    rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
    
    conn, err := amqp.Dial(rabbitmqURL)
    if err != nil {
        log.Fatal("Failed to connect to RabbitMQ:", err)
    }
    
    log.Println("RabbitMQ connected")
    return conn
}

func (s *server) publishEvent(eventType string, data interface{}) error {
    ch, err := s.rabbitmq.Channel()
    if err != nil {
        return err
    }
    defer ch.Close()
    
    err = ch.ExchangeDeclare(
        "events_exchange",
        "direct",
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return err
    }
    
    body, err := json.Marshal(data)
    if err != nil {
        return err
    }
    
    err = ch.Publish(
        "events_exchange",
        eventType,
        false,
        false,
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
    
    return err
}

// CreateUser
func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
    user := bson.M{
        "first_name": req.FirstName,
        "last_name":  req.LastName,
        "email":      req.Email,
        "role":       req.Role,
        "created_at": time.Now().Format(time.RFC3339),
    }
    _, err := s.col.InsertOne(ctx, user)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "DB insert error: %v", err)
    }
    return &pb.UserResponse{
        Id:        0, // Si usas int32, deberás mapear el ObjectID a int32 o cambiar el proto a string
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Email:     req.Email,
        Role:      req.Role,
        CreatedAt: user["created_at"].(string),
    }, nil
}

// GetUser
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
    var user bson.M
    err := s.col.FindOne(ctx, bson.M{"id": req.Id}).Decode(&user)
    if err == mongo.ErrNoDocuments {
        return nil, status.Errorf(codes.NotFound, "Usuario no encontrado")
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, "DB error: %v", err)
    }
    return &pb.UserResponse{
        Id:        req.Id,
        FirstName: user["first_name"].(string),
        LastName:  user["last_name"].(string),
        Email:     user["email"].(string),
        Role:      user["role"].(string),
        CreatedAt: user["created_at"].(string),
    }, nil
}

// UpdateUser
func (s *server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
    update := bson.M{
        "$set": bson.M{
            "first_name": req.FirstName,
            "last_name":  req.LastName,
            "email":      req.Email,
        },
    }
    var user bson.M
    err := s.col.FindOneAndUpdate(ctx, bson.M{"id": req.Id}, update).Decode(&user)
    if err == mongo.ErrNoDocuments {
        return nil, status.Errorf(codes.NotFound, "Usuario no encontrado")
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, "DB error: %v", err)
    }
    return &pb.UserResponse{
        Id:        req.Id,
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Email:     req.Email,
        Role:      user["role"].(string),
        CreatedAt: user["created_at"].(string),
    }, nil
}

// DeleteUser
func (s *server) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
    res, err := s.col.DeleteOne(ctx, bson.M{"id": req.Id})
    if err != nil {
        return nil, status.Errorf(codes.Internal, "DB error: %v", err)
    }
    if res.DeletedCount == 0 {
        return nil, status.Errorf(codes.NotFound, "Usuario no encontrado")
    }
    return &pb.DeleteUserResponse{Message: "Usuario eliminado"}, nil
}

// ListUsers
func (s *server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
    filter := bson.M{}
    if req.Email != "" {
        filter["email"] = req.Email
    }
    if req.Name != "" {
        filter["first_name"] = req.Name // O ajusta según tu modelo
    }
    cur, err := s.col.Find(ctx, filter)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "DB error: %v", err)
    }
    defer cur.Close(ctx)

    var users []*pb.UserResponse
    for cur.Next(ctx) {
        var user bson.M
        if err := cur.Decode(&user); err == nil {
            users = append(users, &pb.UserResponse{
                Id:        int32(user["id"].(int)), // Ajusta según tu modelo
                FirstName: user["first_name"].(string),
                LastName:  user["last_name"].(string),
                Email:     user["email"].(string),
                Role:      user["role"].(string),
                CreatedAt: user["created_at"].(string),
            })
        }
    }
    return &pb.ListUsersResponse{Users: users}, nil
}

func main() {
    port := getEnv("PORT", "50051")
    
    db := initDB()
    defer db.Close()
    
    rabbitmq := initRabbitMQ()
    defer rabbitmq.Close()

    // Inicializa Mongo si usas la colección
    mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal("Error connecting to MongoDB:", err)
    }
    col := mongoClient.Database("users_db").Collection("users")
    
    lis, err := net.Listen("tcp", ":"+port)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterUserServiceServer(s, &server{
        db:      db,
        rabbitmq: rabbitmq,
        col:     col,
    })
    
    log.Printf("Users service listening on port %s", port)
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
