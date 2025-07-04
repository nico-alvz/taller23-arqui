package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "billing-service/pb"
)

//=====================================================================
// CONFIG & DB SETUP
//=====================================================================

var db *sql.DB // global connection pool

// dbConfig holds database connection parameters from env vars.
type dbConfig struct {
	Host string
	Port string
	Name string
	User string
	Pass string
}

func getenv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getDBConfig() dbConfig {
	return dbConfig{
		Host: getenv("DB_HOST", "localhost"),
		Port: getenv("DB_PORT", "3306"),
		Name: getenv("DB_NAME", "billing_db"),
		User: getenv("DB_USER", "root"),
		Pass: getenv("DB_PASSWORD", "password"),
	}
}

func (c dbConfig) dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		c.User, c.Pass, c.Host, c.Port, c.Name)
}

func initDB() error {
	const query = `CREATE TABLE IF NOT EXISTS invoices (
        id INT AUTO_INCREMENT PRIMARY KEY,
        user_id INT NOT NULL,
        amount DECIMAL(10,2) NOT NULL,
        issue_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        payment_date TIMESTAMP NULL,
        status ENUM('Pendiente','Pagado','Vencido') NOT NULL DEFAULT 'Pendiente',
        deleted_at TIMESTAMP NULL,
        INDEX idx_user_id (user_id),
        INDEX idx_status (status),
        INDEX idx_deleted (deleted_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
	_, err := db.Exec(query)
	return err
}

//=====================================================================
// AUTH & UTILITIES
//=====================================================================

const (
	mdUserID = "user_id"        // authenticated user id (string/int64)
	mdRole   = "role"           // "admin" or "client"
	mdTarget = "target_user_id" // optional: admin listing invoices of another user
)

type authCtx struct {
	userID int64
	role   string // "admin" | "client"
}

func getAuth(ctx context.Context) (authCtx, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return authCtx{}, status.Error(codes.Unauthenticated, "metadata missing: user not logged in")
	}

	uidVals := md.Get(mdUserID)
	roleVals := md.Get(mdRole)
	if len(uidVals) == 0 || len(roleVals) == 0 {
		return authCtx{}, status.Error(codes.Unauthenticated, "user not logged in")
	}

	uid, err := strconv.ParseInt(uidVals[0], 10, 64)
	if err != nil {
		return authCtx{}, status.Error(codes.Unauthenticated, "invalid user_id metadata")
	}

	role := roleVals[0]
	if role != "admin" && role != "client" {
		return authCtx{}, status.Error(codes.PermissionDenied, "unknown role")
	}
	return authCtx{userID: uid, role: role}, nil
}

func requireAdmin(a authCtx) error {
	if a.role != "admin" {
		return status.Error(codes.PermissionDenied, "admin privileges required")
	}
	return nil
}

//=====================================================================
// DOMAIN & MAPPING
//=====================================================================

type invoiceRow struct {
	ID          int64
	UserID      int64
	Amount      float64
	IssueDate   time.Time
	PaymentDate sql.NullTime
	Status      string
}

func mapStatusEnumToString(s pb.InvoiceStatus) (string, error) {
	switch s {
	case pb.InvoiceStatus_PENDIENTE:
		return "Pendiente", nil
	case pb.InvoiceStatus_PAGADO:
		return "Pagado", nil
	case pb.InvoiceStatus_VENCIDO:
		return "Vencido", nil
	case pb.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED:
		return "", errors.New("unspecified status")
	default:
		return "", fmt.Errorf("unknown status enum: %v", s)
	}
}

func mapStatusStringToEnum(s string) pb.InvoiceStatus {
	switch s {
	case "Pendiente":
		return pb.InvoiceStatus_PENDIENTE
	case "Pagado":
		return pb.InvoiceStatus_PAGADO
	case "Vencido":
		return pb.InvoiceStatus_VENCIDO
	default:
		return pb.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED
	}
}

func rowToProto(r invoiceRow) *pb.Invoice {
	var payTS *timestamppb.Timestamp
	if r.PaymentDate.Valid {
		payTS = timestamppb.New(r.PaymentDate.Time)
	}
	return &pb.Invoice{
		Id:          r.ID,
		UserId:      r.UserID,
		Status:      mapStatusStringToEnum(r.Status),
		Amount:      int64(math.Round(r.Amount * 100)), // cents
		IssueDate:   timestamppb.New(r.IssueDate),
		PaymentDate: payTS,
	}
}

//=====================================================================
// GRPC SERVICE IMPLEMENTATION
//=====================================================================

type billingServer struct {
	pb.UnimplementedBillingServiceServer
}

// CreateInvoice implements business rules for invoice creation.
func (s *billingServer) CreateInvoice(ctx context.Context, req *pb.CreateInvoiceRequest) (*pb.CreateInvoiceResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireAdmin(auth); err != nil {
		return nil, err
	}

	// Validate amount > 0 (positive integer)
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	statusStr := "Pendiente"
	if req.Status != pb.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED {
		statusStr, err = mapStatusEnumToString(req.Status)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	amountFloat := float64(req.Amount) / 100.0 // convert cents to decimal

	res, err := db.ExecContext(ctx, `INSERT INTO invoices (user_id, amount, status) VALUES (?,?,?)`, req.UserId, amountFloat, statusStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create invoice: %v", err)
	}
	id, _ := res.LastInsertId()

	inv, err := fetchInvoiceByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve invoice: %v", err)
	}
	return &pb.CreateInvoiceResponse{Invoice: inv}, nil
}

// GetInvoiceById returns invoice if requester authorized.
func (s *billingServer) GetInvoiceById(ctx context.Context, req *pb.GetInvoiceByIdRequest) (*pb.GetInvoiceByIdResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}

	inv, err := fetchInvoiceByID(ctx, req.Id)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "invoice not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if auth.role != "admin" && inv.UserId != auth.userID {
		return nil, status.Error(codes.PermissionDenied, "not authorized to view this invoice")
	}

	return &pb.GetInvoiceByIdResponse{Invoice: inv}, nil
}

// UpdateInvoiceState allows only admins to change status.
func (s *billingServer) UpdateInvoiceState(ctx context.Context, req *pb.UpdateInvoiceStateRequest) (*pb.UpdateInvoiceStateResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireAdmin(auth); err != nil {
		return nil, err
	}

	statusStr, err := mapStatusEnumToString(req.NewStatus)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var query string
	var args []interface{}
	if statusStr == "Pagado" {
		query = `UPDATE invoices SET status = ?, payment_date = NOW() WHERE id = ? AND deleted_at IS NULL`
		args = []interface{}{statusStr, req.Id}
	} else {
		query = `UPDATE invoices SET status = ? WHERE id = ? AND deleted_at IS NULL`
		args = []interface{}{statusStr, req.Id}
	}

	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, status.Error(codes.NotFound, "invoice not found or deleted")
	}

	inv, err := fetchInvoiceByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetch error: %v", err)
	}
	return &pb.UpdateInvoiceStateResponse{Invoice: inv}, nil
}

// DeleteInvoice performs soft delete; only admins.
func (s *billingServer) DeleteInvoice(ctx context.Context, req *pb.DeleteInvoiceRequest) (*emptypb.Empty, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireAdmin(auth); err != nil {
		return nil, err
	}

	var statusStr string
	err = db.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id = ? AND deleted_at IS NULL`, req.Id).Scan(&statusStr)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "invoice not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if statusStr == "Pagado" {
		return nil, status.Error(codes.FailedPrecondition, "cannot delete a paid invoice")
	}

	_, err = db.ExecContext(ctx, `UPDATE invoices SET deleted_at = NOW() WHERE id = ?`, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete failed: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListInvoicesByUser lists invoices; role determines scope.
func (s *billingServer) ListInvoicesByUser(ctx context.Context, req *pb.ListInvoicesByUserRequest) (*pb.ListInvoicesByUserResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}

	var targetUserID int64
	if auth.role == "admin" {
		md, _ := metadata.FromIncomingContext(ctx)
		tgt := md.Get(mdTarget)
		if len(tgt) > 0 {
			targetUserID, err = strconv.ParseInt(tgt[0], 10, 64)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid target_user_id")
			}
		} else {
			targetUserID = auth.userID // default to own if not specified
		}
	} else {
		targetUserID = auth.userID
	}

	query := `SELECT id, user_id, amount, issue_date, payment_date, status FROM invoices WHERE deleted_at IS NULL AND user_id = ?`
	args := []interface{}{targetUserID}

	if req.StatusFilter != nil && *req.StatusFilter != pb.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED {
		statusStr, _ := mapStatusEnumToString(*req.StatusFilter)
		query += " AND status = ?"
		args = append(args, statusStr)
	}

	query += " ORDER BY issue_date DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	invoices := make([]*pb.Invoice, 0)
	for rows.Next() {
		var r invoiceRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.Amount, &r.IssueDate, &r.PaymentDate, &r.Status); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		invoices = append(invoices, rowToProto(r))
	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "row error: %v", err)
	}

	return &pb.ListInvoicesByUserResponse{Invoices: invoices}, nil
}

//=====================================================================
// HELPERS
//=====================================================================

func fetchInvoiceByID(ctx context.Context, id int64) (*pb.Invoice, error) {
	var r invoiceRow
	err := db.QueryRowContext(ctx, `SELECT id, user_id, amount, issue_date, payment_date, status FROM invoices WHERE id = ? AND deleted_at IS NULL`, id).
		Scan(&r.ID, &r.UserID, &r.Amount, &r.IssueDate, &r.PaymentDate, &r.Status)
	if err != nil {
		return nil, err
	}
	return rowToProto(r), nil
}

//=====================================================================
// MAIN
//=====================================================================

func main() {
	cfg := getDBConfig()
	var err error
	db, err = sql.Open("mysql", cfg.dsn())
	if err != nil {
		log.Fatalf("cannot open DB connection: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("cannot ping DB: %v", err)
	}
	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err = initDB(); err != nil {
		log.Fatalf("cannot init DB: %v", err)
	}
	log.Println("Billing database initialized")

	lis, err := net.Listen("tcp", ":"+getenv("PORT", "50052"))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterBillingServiceServer(grpcServer, &billingServer{})

	log.Printf("gRPC BillingService listening on %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
