package main

/*
Email Microservice with gRPC support (StreamFlow)
------------------------------------------------
• gRPC API (port 50057 by default)
    - SendInvoiceEmail → envía notificación de factura actualizada.
• HTTP API (Gin) conservado para compatibilidad (puerto 50058) – opcional.
• RabbitMQ consumer sigue activo para eventos user.created, invoice.updated,
  password.updated.
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Make sure this import path matches the actual location of your generated pb.go file.
	pb "email-service/pb"
)

// -------------------- gRPC placeholder types (remove if pb is available) --------------------

type SendInvoiceEmailRequest struct {
	UserEmail string
	InvoiceId int64
	Amount    string
	Status    string
}

type SendInvoiceEmailResponse struct {
	Success bool
}

// -------------------- Config helpers --------------------

type smtpConfig struct{ Host, Port, User, Pass string }

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func smtpCfg() smtpConfig {
	return smtpConfig{
		Host: getenv("SMTP_HOST", "smtp.gmail.com"),
		Port: getenv("SMTP_PORT", "587"),
		User: getenv("SMTP_USER", "streamflow.app.2024@gmail.com"),
		Pass: getenv("SMTP_PASSWORD", "your_app_password"),
	}
}

var rabbitURL = getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

// -------------------- Email sender (mock) --------------------

func sendEmail(to, subject, body string) bool {
	log.Printf("📧 EMAIL ENVIADO → %s | %s\n%s", to, subject, body)
	return true
}

// -------------------- gRPC implementation --------------------

type emailServer struct {
	pb.UnimplementedEmailServiceServer
}

func (s *emailServer) SendInvoiceEmail(ctx context.Context, req *pb.SendInvoiceEmailRequest) (*pb.SendInvoiceEmailResponse, error) {
	if req.UserEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "user_email obligatorio")
	}

	subj := fmt.Sprintf("Actualización de Factura #%d", req.InvoiceId)
	body := fmt.Sprintf(`<html><body><h2>Actualización de Factura</h2><ul><li><b>Factura #:</b> %d</li><li><b>Monto:</b> $%s</li><li><b>Estado:</b> %s</li></ul></body></html>`, req.InvoiceId, req.Amount, req.Status)
	ok := sendEmail(req.UserEmail, subj, body)
	return &pb.SendInvoiceEmailResponse{Success: ok}, nil
}

func (s *emailServer) SendPasswordUpdatedEmail(ctx context.Context, req *pb.PasswordEmailRequest) (*pb.EmailResponse, error) {
	// Implement your logic here or return a stub response
	return &pb.EmailResponse{
		Success: true,
		Message: "Password update email sent (stub)",
	}, nil
}

func (s *emailServer) SendWelcomeEmail(ctx context.Context, req *pb.WelcomeEmailRequest) (*pb.EmailResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email obligatorio")
	}
	subj := "¡Bienvenido a StreamFlow!"
	body := fmt.Sprintf(`<html><body><h2>¡Bienvenido, %s!</h2></body></html>`, req.Name)
	ok := sendEmail(req.Email, subj, body)
	return &pb.EmailResponse{
		Success: ok,
		Message: "Welcome email sent",
	}, nil
}

// -------------------- RabbitMQ consumer --------------------

type consumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func newConsumer() (*consumer, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &consumer{conn: conn, ch: ch}, nil
}
func (c *consumer) close() {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *consumer) setup() error {
	if err := c.ch.ExchangeDeclare("events_exchange", "direct", true, false, false, false, nil); err != nil {
		return err
	}
	qdefs := []string{"user_creation_queue", "invoice_update_queue", "password_update_queue"}
	for _, q := range qdefs {
		if _, err := c.ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return err
		}
	}
	binds := []struct{ q, key string }{
		{"user_creation_queue", "user.created"},
		{"invoice_update_queue", "invoice.updated"},
		{"password_update_queue", "password.updated"},
	}
	for _, b := range binds {
		if err := c.ch.QueueBind(b.q, b.key, "events_exchange", false, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *consumer) start(ctx context.Context) {
	go consume(ctx, c, "user_creation_queue", handleUserCreated)
	go consume(ctx, c, "invoice_update_queue", handleInvoiceUpdated)
	go consume(ctx, c, "password_update_queue", handlePasswordUpdated)
	log.Println("RabbitMQ consumers running …")
}

func consume(ctx context.Context, c *consumer, queue string, h func(amqp.Delivery)) {
	msgs, _ := c.ch.Consume(queue, "", false, false, false, false, nil)
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			h(d)
		}
	}
}

// -------------------- Message handlers --------------------

func handleUserCreated(d amqp.Delivery) {
	var m struct{ Email, Name string }
	if err := json.Unmarshal(d.Body, &m); err != nil {
		log.Printf("malformed user.created: %v", err)
		_ = d.Nack(false, false)
		return
	}
	subj := "¡Bienvenido a StreamFlow!"
	body := fmt.Sprintf(`<html><body><h2>¡Bienvenido, %s!</h2></body></html>`, m.Name)
	sendEmail(m.Email, subj, body)
	_ = d.Ack(false)
}

func handleInvoiceUpdated(d amqp.Delivery) {
	var m struct {
		UserEmail      string `json:"user_email"`
		InvoiceID      int64  `json:"invoice_id"`
		Amount, Status string
	}
	if err := json.Unmarshal(d.Body, &m); err != nil {
		log.Printf("malformed invoice.updated: %v", err)
		_ = d.Nack(false, false)
		return
	}
	subj := fmt.Sprintf("Actualización de Factura #%d", m.InvoiceID)
	body := fmt.Sprintf(`<html><body><ul><li>#%d</li><li>$%s</li><li>%s</li></ul></body></html>`, m.InvoiceID, m.Amount, m.Status)
	sendEmail(m.UserEmail, subj, body)
	_ = d.Ack(false)
}

func handlePasswordUpdated(d amqp.Delivery) {
	var m struct{ UserEmail, UserName string }
	if err := json.Unmarshal(d.Body, &m); err != nil {
		log.Printf("malformed password.updated: %v", err)
		_ = d.Nack(false, false)
		return
	}
	subj := "Contraseña Actualizada - StreamFlow"
	body := fmt.Sprintf(`<html><body><p>Hola %s, tu contraseña se actualizó.</p></body></html>`, m.UserName)
	sendEmail(m.UserEmail, subj, body)
	_ = d.Ack(false)
}

// -------------------- HTTP fallback --------------------

func httpRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "email"}) })
	return r
}

// -------------------- MAIN --------------------

func main() {
	// RabbitMQ setup
	cons, err := newConsumer()
	if err != nil {
		log.Fatalf("rabbit: %v", err)
	}
	if err := cons.setup(); err != nil {
		log.Fatalf("rabbit setup: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cons.start(ctx)

	// gRPC server
	grpcPort := getenv("GRPC_PORT", "50057")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	pb.RegisterEmailServiceServer(grpcSrv, &emailServer{})
	go func() {
		log.Printf("gRPC EmailService on :%s", grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	// Optional HTTP server
	httpPort := getenv("HTTP_PORT", "50058")
	httpSrv := &http.Server{Addr: ":" + httpPort, Handler: httpRouter()}
	go func() {
		log.Printf("HTTP health on :%s", httpPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	grpcSrv.GracefulStop()
	_ = httpSrv.Close()
	cons.close()
	time.Sleep(300 * time.Millisecond)
}
