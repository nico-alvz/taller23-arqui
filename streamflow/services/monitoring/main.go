package main

import (
    "context"
    "database/sql"
    "log"
    "net"
    
    "time"

    _ "github.com/go-sql-driver/mysql"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/emptypb"
    "google.golang.org/protobuf/types/known/timestamppb"

    pb "monitoring-service/pb"
)

//================ DB & AUTH HELPERS (re‑uso de funciones del billing) ================

type monDBConfig struct{ Host, Port, Name, User, Pass string }

func monGetCfg() monDBConfig {
    return monDBConfig{
        Host: getenv("MON_DB_HOST", getenv("DB_HOST", "localhost")),
        Port: getenv("MON_DB_PORT", getenv("DB_PORT", "3306")),
        Name: getenv("MON_DB_NAME", "monitoring_db"),
        User: getenv("MON_DB_USER", getenv("DB_USER", "root")),
        Pass: getenv("MON_DB_PASSWORD", getenv("DB_PASSWORD", "password")),
    }
}
func (c monDBConfig) dsn() string {
    return c.User + ":" + c.Pass + "@tcp(" + c.Host + ":" + c.Port + ")/" + c.Name + "?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"
}

var monDB *sql.DB

func initMonDB() error {
    // Dos tablas: actions y errors
    const (
        q1 = `CREATE TABLE IF NOT EXISTS actions (
            id INT AUTO_INCREMENT PRIMARY KEY,
            user_id INT NULL,
            email VARCHAR(255) NULL,
            method VARCHAR(10) NOT NULL,
            url VARCHAR(512) NOT NULL,
            action VARCHAR(256) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
        q2 = `CREATE TABLE IF NOT EXISTS errors (
            id INT AUTO_INCREMENT PRIMARY KEY,
            user_id INT NULL,
            email VARCHAR(255) NULL,
            error_message TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
    )
    if _, err := monDB.Exec(q1); err != nil { return err }
    if _, err := monDB.Exec(q2); err != nil { return err }
    return nil
}

//======================== MONITORING SERVICE IMPLEMENTATION ========================

type monitoringSrv struct{ pb.UnimplementedMonitoringServiceServer }

func (s *monitoringSrv) ListActions(ctx context.Context, _ *emptypb.Empty) (*pb.ListActionsResponse, error) {
    auth, err := getAuth(ctx)
    if err != nil { return nil, err }
    if err := requireAdmin(auth); err != nil { return nil, err }

    rows, err := monDB.QueryContext(ctx, `SELECT id, created_at, user_id, email, method, url, action FROM actions ORDER BY created_at DESC`)
    if err != nil { return nil, status.Errorf(codes.Internal, "query error: %v", err) }
    defer rows.Close()

    var res pb.ListActionsResponse
    for rows.Next() {
        var (
            id int64
            ts time.Time
            uid sql.NullInt64
            email sql.NullString
            method, url, action string
        )
        if err := rows.Scan(&id, &ts, &uid, &email, &method, &url, &action); err != nil {
            return nil, status.Errorf(codes.Internal, "scan error: %v", err)
        }
        res.Actions = append(res.Actions, &pb.ActionLog{
            Id:        id,
            Timestamp: timestamppb.New(ts),
            UserId:    nullInt64(uid),
            Email:     nullString(email),
            Method:    method,
            Url:       url,
            Action:    action,
        })
    }
    if err := rows.Err(); err != nil {
        return nil, status.Errorf(codes.Internal, "row error: %v", err)
    }
    return &res, nil
}

func (s *monitoringSrv) ListErrors(ctx context.Context, _ *emptypb.Empty) (*pb.ListErrorsResponse, error) {
    auth, err := getAuth(ctx)
    if err != nil { return nil, err }
    if err := requireAdmin(auth); err != nil { return nil, err }

    rows, err := monDB.QueryContext(ctx, `SELECT id, created_at, user_id, email, error_message FROM errors ORDER BY created_at DESC`)
    if err != nil { return nil, status.Errorf(codes.Internal, "query error: %v", err) }
    defer rows.Close()

    var res pb.ListErrorsResponse
    for rows.Next() {
        var (
            id int64
            ts time.Time
            uid sql.NullInt64
            email sql.NullString
            errMsg string
        )
        if err := rows.Scan(&id, &ts, &uid, &email, &errMsg); err != nil {
            return nil, status.Errorf(codes.Internal, "scan error: %v", err)
        }
        res.Errors = append(res.Errors, &pb.ErrorLog{
            Id:          id,
            Timestamp:   timestamppb.New(ts),
            UserId:      nullInt64(uid),
            Email:       nullString(email),
            ErrorMessage: errMsg,
        })
    }
    if err := rows.Err(); err != nil {
        return nil, status.Errorf(codes.Internal, "row error: %v", err)
    }
    return &res, nil
}

//======================== UTILIDADES AUXILIARES ========================

func nullInt64(n sql.NullInt64) int64 {
    if n.Valid { return n.Int64 }
    return 0 // cero = vacío según proto3
}
func nullString(s sql.NullString) string {
    if s.Valid { return s.String }
    return ""
}

//======================== MAIN BOOTSTRAP ==============================

func main() {
    cfg := monGetCfg()
    var err error
    monDB, err = sql.Open("mysql", cfg.dsn())
    if err != nil { log.Fatalf("cannot open monitoring DB: %v", err) }
    if err = monDB.Ping(); err != nil { log.Fatalf("cannot ping monitoring DB: %v", err) }
    if err = initMonDB(); err != nil { log.Fatalf("cannot init monitoring schema: %v", err) }
    log.Println("Monitoring DB ready")

    lis, err := net.Listen("tcp", ":"+getenv("MON_PORT", "50053"))
    if err != nil { log.Fatalf("listen: %v", err) }

    grpcServer := grpc.NewServer()
    pb.RegisterMonitoringServiceServer(grpcServer, &monitoringSrv{})

    log.Printf("gRPC MonitoringService listening on %s", lis.Addr())
    if err := grpcServer.Serve(lis); err != nil { log.Fatalf("serve: %v", err) }
}




type monitoringServer struct {
    pb.UnimplementedMonitoringServiceServer
}

// Implementa ListActions
func (s *monitoringServer) ListActions(ctx context.Context, _ *emptypb.Empty) (*pb.ListActionsResponse, error) {
    actions := []*pb.ActionLog{
        {
            Id:        1,
            Timestamp: timestamppb.New(time.Now()),
            UserId:    123,
            Email:     "user@example.com",
            Method:    "GET",
            Url:       "/api/resource",
            Action:    "viewed",
        },
    }
    return &pb.ListActionsResponse{Actions: actions}, nil
}

// Implementa ListErrors
func (s *monitoringServer) ListErrors(ctx context.Context, _ *emptypb.Empty) (*pb.ListErrorsResponse, error) {
    errors := []*pb.ErrorLog{
        {
            Id:           1,
            Timestamp:    timestamppb.New(time.Now()),
            UserId:       123,
            Email:        "user@example.com",
            ErrorMessage: "Something went wrong",
        },
    }
    return &pb.ListErrorsResponse{Errors: errors}, nil
}

// Removed duplicate grpcServer initialization and registration.
