package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq" // PostgreSQL driver
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Configuration
var (
	SECRET_KEY                = os.Getenv("JWT_SECRET_KEY")
	ALGORITHM                 = "HS256" // Should match the signing method
	ACCESS_TOKEN_EXPIRE_MINUTES = 1440

	DB_HOST     = os.Getenv("DB_HOST")
	DB_PORT     = os.Getenv("DB_PORT")
	DB_NAME     = os.Getenv("DB_NAME")
	DB_USER     = os.Getenv("DB_USER")
	DB_PASSWORD = os.Getenv("DB_PASSWORD")

	RABBITMQ_HOST = os.Getenv("RABBITMQ_HOST")
	RABBITMQ_USER = os.Getenv("RABBITMQ_USER")
	RABBITMQ_PASS = os.Getenv("RABBITMQ_PASS")
	RABBITMQ_QUEUE = os.Getenv("RABBITMQ_QUEUE")
)

// Models (Go Structs with JSON tags)
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	CurrentPassword     string `json:"current_password"`
	NewPassword         string `json:"new_password"`
	ConfirmNewPassword string `json:"confirm_new_password"`
}

type UserResponse struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	User       UserResponse `json:"user"`
	AccessToken string       `json:"access_token"`
	TokenType  string       `json:"token_type"`
}

type ErrorResponse struct {
	Detail string `json:"detail"`
}

// Custom JWT Claims
type Claims struct {
	UserID int64  `json:"sub"` // Standard "sub" claim (subject)
	Email  string `json:"email"`
	Role   string `json:"role"`
	JTI    string `json:"jti"` // JWT ID for blacklist
	jwt.RegisteredClaims
}

// AuthService holds dependencies like DB connection
type AuthService struct {
	db *sql.DB
	// RabbitMQ connection is handled per-publish for simplicity,
	// but a persistent connection is better in production.
}

// --- Database Functions ---

func getDBConnection(as *AuthService) *sql.DB {
	// Use the connection pool managed by sql.Open
	if as.db == nil {
		log.Fatal("Database connection pool is not initialized")
	}
	return as.db
}

func initDB(db *sql.DB) error {
	// Check if SECRET_KEY is set
	if SECRET_KEY == "" {
		log.Println("⚠️ JWT_SECRET_KEY not set, using default. Set this in production!")
		SECRET_KEY = "streamflow_secret_key_2024" // Default key
	}
	if RABBITMQ_HOST == "" {
		log.Println("⚠️ RABBITMQ_HOST not set, using default 'rabbitmq'")
		RABBITMQ_HOST = "rabbitmq"
	}
	if RABBITMQ_USER == "" {
		log.Println("⚠️ RABBITMQ_USER not set, using default 'guest'")
		RABBITMQ_USER = "guest"
	}
	if RABBITMQ_PASS == "" {
		log.Println("⚠️ RABBITMQ_PASS not set, using default 'guest'")
		RABBITMQ_PASS = "guest"
	}
	if RABBITMQ_QUEUE == "" {
		log.Println("⚠️ RABBITMQ_QUEUE not set, using default 'monitoring'")
		RABBITMQ_QUEUE = "monitoring"
	}


	// DSN (Data Source Name) for PostgreSQL
	// Example: "host=localhost port=5432 user=postgres password=password dbname=auth_db sslmode=disable"
	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)

	var err error
	db, err = sql.Open("postgres", dbinfo)
	if err != nil {
		return fmt.Errorf("error opening database connection: %w", err)
	}

	// Set connection pool settings (optional but recommended)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}
	log.Println("✅ Connected to PostgreSQL database")

	// Create tables if they don't exist
	createUsersTableSQL := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			first_name VARCHAR(50) NOT NULL,
			last_name VARCHAR(50) NOT NULL,
			email VARCHAR(100) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL CHECK (role IN ('Administrador', 'Cliente')),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE NULL
		);
	`
	_, err = db.Exec(createUsersTableSQL)
	if err != nil {
		return fmt.Errorf("error creating users table: %w", err)
	}

	createBlacklistTableSQL := `
		CREATE TABLE IF NOT EXISTS token_blacklist (
			id SERIAL PRIMARY KEY,
			jti VARCHAR(255) NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err = db.Exec(createBlacklistTableSQL)
	if err != nil {
		return fmt.Errorf("error creating token_blacklist table: %w", err)
	}

	// Insert default admin user if not exists (ON CONFLICT DO NOTHING)
	adminPasswordHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing admin password: %w", err)
	}

	insertAdminSQL := `
		INSERT INTO users (first_name, last_name, email, password, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO NOTHING;
	`
	_, err = db.Exec(insertAdminSQL, "Admin", "StreamFlow", "admin@streamflow.com", string(adminPasswordHash), "Administrador")
	if err != nil {
		return fmt.Errorf("error inserting default admin user: %w", err)
	}

	log.Println("✅ Database initialized successfully.")
	return nil
}

func (s *AuthService) isTokenBlacklisted(jti string) (bool, error) {
	db := getDBConnection(s)
	var id int
	err := db.QueryRow("SELECT id FROM token_blacklist WHERE jti = $1", jti).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Not blacklisted
	}
	if err != nil {
		return false, fmt.Errorf("error checking token blacklist: %w", err)
	}
	return true, nil // Blacklisted
}

// --- JWT Functions ---

func createAccessToken(userID int, email string, role string) (string, string, error) {
	// JTI (JWT ID) should be unique per token
	jti := fmt.Sprintf("%d_%d", userID, time.Now().UnixNano())

	expirationTime := time.Now().Add(time.Minute * time.Duration(ACCESS_TOKEN_EXPIRE_MINUTES))

	claims := &Claims{
		UserID: int64(userID),
		Email:  email,
		Role:   role,
		JTI:    jti,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   strconv.Itoa(userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", "", fmt.Errorf("error signing token: %w", err)
	}

	return tokenString, jti, nil
}

func verifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(SECRET_KEY), nil
	})

	if err != nil {
		// Check for specific errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("token expirado") // Match Python error message
		}
		return nil, fmt.Errorf("token inválido: %w", err) // Match Python error message
	}

	if !token.Valid {
		return nil, fmt.Errorf("token inválido")
	}

	return claims, nil
}

// --- Authentication Middleware/Helper ---

// getCurrentUser extracts and verifies the token from the request header
func (s *AuthService) getCurrentUser(r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("Authorization header required")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, fmt.Errorf("Invalid Authorization header format")
	}

	tokenString := parts[1]
	claims, err := verifyToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("Token verification failed: %w", err)
	}

	blacklisted, err := s.isTokenBlacklisted(claims.JTI)
	if err != nil {
		log.Printf("Error checking blacklist for JTI %s: %v", claims.JTI, err)
		return nil, fmt.Errorf("Error interno al verificar token") // Hide internal error detail from user
	}
	if blacklisted {
		return nil, fmt.Errorf("Token invalidado")
	}

	return claims, nil
}

// requireAuth is a middleware/wrapper for protected handlers
func (s *AuthService) requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := s.getCurrentUser(r)
		if err != nil {
			// Convert error to HTTP response
			statusCode := http.StatusUnauthorized
			detail := "Autenticación requerida" // Default message
			if strings.Contains(err.Error(), "Token expirado") || strings.Contains(err.Error(), "Token inválido") || strings.Contains(err.Error(), "Token invalidado") {
				statusCode = http.StatusUnauthorized
				detail = err.Error() // Use specific token error message
			} else if strings.Contains(err.Error(), "Authorization header") {
                 statusCode = http.StatusBadRequest // Bad request if header format is wrong
                 detail = err.Error()
            } else {
                log.Printf("Unexpected authentication error: %v", err)
                statusCode = http.StatusInternalServerError // Internal error for other issues
                detail = "Error interno de autenticación"
            }

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(ErrorResponse{Detail: detail})
			return
		}
		// Add claims to request context for handler to access
		ctx := context.WithValue(r.Context(), "userClaims", claims)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

// requireAdmin is a middleware/wrapper for admin-only handlers
func (s *AuthService) requireAdmin(handler http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value("userClaims").(*Claims)
		if !ok || claims.Role != "Administrador" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(ErrorResponse{Detail: "No tiene permisos para esta acción"})
			return
		}
		handler.ServeHTTP(w, r)
	})
}


// --- RabbitMQ Publisher ---

func publishEvent(eventType string, payload map[string]interface{}) {
	// Use the connection URL format: amqp://user:password@host:port/vhost
	// vhost is usually "/" by default
	rabbitMQURL := fmt.Sprintf("amqp://%s:%s@%s/", RABBITMQ_USER, RABBITMQ_PASS, RABBITMQ_HOST)

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Printf("❌ Failed to connect to RabbitMQ: %v", err)
		// In a real application, implement retry logic here
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("❌ Failed to open a channel: %v", err)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		RABBITMQ_QUEUE, // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		log.Printf("❌ Failed to declare a queue: %v", err)
		return
	}

	messageBody, err := json.Marshal(map[string]interface{}{
		"type":      eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339), // Use RFC3339 for consistent timestamp format
		"data":      payload,
	})
	if err != nil {
		log.Printf("❌ Failed to marshal event message: %v", err)
		return
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Persistent message
			ContentType:  "application/json",
			Body:         messageBody,
		})
	if err != nil {
		log.Printf("❌ Failed to publish a message: %v", err)
		return
	}

	log.Printf("📤 Event published to RabbitMQ queue '%s': %s", RABBITMQ_QUEUE, eventType)
}


// --- HTTP Handlers ---

func (s *AuthService) loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var loginData LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Invalid request payload"})
		return
	}

	db := getDBConnection(s)
	var user UserResponse
	var hashedPassword string
	var deletedAt sql.NullTime // Use sql.NullTime for nullable timestamp

	row := db.QueryRow(`
		SELECT id, first_name, last_name, email, password, role, created_at, deleted_at
		FROM users
		WHERE email = $1
	`, loginData.Email)

	err := row.Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&hashedPassword,
		&user.Role,
		&user.CreatedAt,
		&deletedAt, // Scan into sql.NullTime
	)

	if err == sql.ErrNoRows {
		log.Printf("⚠️ Login failed: User not found for email %s", loginData.Email)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Credenciales inválidas"})
		return
	}
	if err != nil {
		log.Printf("❌ Database error during login: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Error en el servidor"})
		return
	}

	// Check if user is soft-deleted
	if deletedAt.Valid {
		log.Printf("⚠️ Login failed: User %d (%s) is soft-deleted", user.ID, user.Email)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Credenciales inválidas"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginData.Password)); err != nil {
		log.Printf("⚠️ Login failed: Invalid password for user %d (%s)", user.ID, user.Email)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Credenciales inválidas"})
		return
	}

	// Generate token
	tokenString, jti, err := createAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		log.Printf("❌ Error creating access token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Error en el servidor al generar token"})
		return
	}

	// Publish monitoring event
	go publishEvent("USER_LOGIN", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
	})

	// Respond with token and user data
	loginResponse := LoginResponse{
		User:       user,
		AccessToken: tokenString,
		TokenType:  "bearer",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResponse)
	log.Printf("✅ User logged in: %d (%s)", user.ID, user.Email)
}

func (s *AuthService) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userIDStr := vars["user_id"]
	targetUserID, err := strconv.Atoi(userIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Invalid user ID format"})
		return
	}

	claims := r.Context().Value("userClaims").(*Claims) // Get claims from context

	// Authorization check
	if claims.Role != "Administrador" && int64(targetUserID) != claims.UserID {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "No tiene permisos para esta acción"})
		return
	}

	var passwordData ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&passwordData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Invalid request payload"})
		return
	}

	if passwordData.NewPassword != passwordData.ConfirmNewPassword {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Las contraseñas no coinciden"})
		return
	}

	db := getDBConnection(s)
	var hashedPassword string
	var deletedAt sql.NullTime

	// Get user data to verify current password if changing own password
	row := db.QueryRow(`
		SELECT password, deleted_at
		FROM users
		WHERE id = $1
	`, targetUserID)

	err = row.Scan(&hashedPassword, &deletedAt)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Usuario no encontrado"})
		return
	}
	if err != nil {
		log.Printf("❌ Database error getting user for password change: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Error interno al cambiar la contraseña"})
		return
	}

	// Check if user is soft-deleted
	if deletedAt.Valid {
		w.WriteHeader(http.StatusNotFound) // Treat soft-deleted as not found for this operation
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Usuario no encontrado"})
		return
	}

	// Verify current password only if the user is changing their own password
	if int64(targetUserID) == claims.UserID {
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(passwordData.CurrentPassword)); err != nil {
			w.WriteHeader(http.StatusBadRequest) // Bad request because the input (current password) is wrong
			json.NewEncoder(w).Encode(ErrorResponse{Detail: "Contraseña actual incorrecta"})
			return
		}
	}
	// If admin is changing another user's password, current password is not required/checked here.
	// The Python code's logic seems to imply current_password is *always* sent in the request body
	// but only checked if it's the same user. This Go code follows that.

	// Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordData.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Error hashing new password: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Error interno al procesar la contraseña"})
		return
	}

	// Update password in DB
	_, err = db.Exec(`
		UPDATE users
		SET password = $1
		WHERE id = $2
	`, string(newHashedPassword), targetUserID)

	if err != nil {
		log.Printf("❌ Database error updating password: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Detail: "Error interno al cambiar la contraseña"})
		return
	}

	// Publish monitoring event
	go publishEvent("USER_PWD_CHANGED", map[string]interface{}{
		"user_id": claims.UserID, // ID of the user performing the action
		"email":   claims.Email,
		"role":    claims.Role,
		"target_user_id": targetUserID, // ID of the user whose password was changed
	})


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Contraseña actualizada exitosamente"})
	log.Printf("✅ Password updated for user ID %d by user ID %d", targetUserID, claims.UserID)
}

func (s *AuthService) logoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims := r.Context().Value("userClaims").(*Claims) // Get claims from context

	db := getDBConnection(s)

	// Add token JTI to blacklist
	_, err := db.Exec(`
		INSERT INTO token_blacklist (jti, user_id)
		VALUES ($1, $2)
	`, claims.JTI, claims.UserID)

	if err != nil {
		// Check for unique constraint violation (already blacklisted)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			log.Printf("⚠️ Token %s already blacklisted for user %d", claims.JTI, claims.UserID)
			// Although already blacklisted, we can still return success for idempotency
		} else {
			log.Printf("❌ Database error blacklisting token: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Detail: "Error interno al cerrar sesión"})
			return
		}
	}

	// Publish monitoring event
	go publishEvent("USER_LOGOUT", map[string]interface{}{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Sesión cerrada exitosamente"})
	log.Printf("✅ User logged out: %d (%s)", claims.UserID, claims.Email)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "auth"})
}


func main() {
	// Load configuration from environment variables
	// Defaults are handled in initDB for some, but DB connection string needs them here
	// Ensure required env vars are set, or provide sensible defaults if appropriate
	if DB_HOST == "" { DB_HOST = "localhost" }
	if DB_PORT == "" { DB_PORT = "5432" }
	if DB_NAME == "" { DB_NAME = "auth_db" }
	if DB_USER == "" { DB_USER = "postgres" }
	if DB_PASSWORD == "" { DB_PASSWORD = "password" }
	// RabbitMQ and SECRET_KEY defaults/checks are in initDB

	// Initialize database connection pool and schema
	var db *sql.DB
	err := initDB(db) // Pass db pointer to be initialized
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	// The db variable is now initialized by initDB

	authService := &AuthService{db: db}

	// Setup HTTP router
	router := mux.NewRouter()

	// Public endpoint
	router.HandleFunc("/auth/login", authService.loginHandler).Methods("POST")
	router.HandleFunc("/health", healthCheckHandler).Methods("GET") // Health check is also public

	// Protected endpoints
	// Use requireAuth middleware for endpoints requiring authentication
	router.HandleFunc("/auth/users/{user_id:[0-9]+}", authService.requireAuth(authService.changePasswordHandler)).Methods("PATCH")
	router.HandleFunc("/auth/logout", authService.requireAuth(authService.logoutHandler)).Methods("POST")

	// Start the HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001" // Default port
	}

	log.Printf("🚀 Auth Service started on HTTP port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

To Run This Code:

    Save the code as main.go in a directory (e.g., auth-service).
    Run go mod init auth-service (replace auth-service with your module name).
    Run go get github.com/gorilla/mux github.com/lib/pq golang.org/x/crypto/bcrypt github.com/golang-jwt/jwt/v5 github.com/rabbitmq/amqp091-go.
    Set environment variables for your PostgreSQL and RabbitMQ connections, and the JWT secret key:

export DB_HOST=localhost
    export DB_PORT=5432
    export DB_NAME=auth_db
    export DB_USER=postgres
    export DB_PASSWORD=password
    export RABBITMQ_HOST=rabbitmq # or your RabbitMQ host
    export RABBITMQ_USER=admin
    export RABBITMQ_PASS=password
    export RABBITMQ_QUEUE=monitoring
    export JWT_SECRET_KEY=your_very_secret_key_here # IMPORTANT: Change this!
    export PORT=8001 # Optional, defaults to 8001