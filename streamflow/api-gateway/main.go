package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"
    "bytes"
    "io"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/status"
    "api-gateway/pb"
)

type Claims struct {
    UserID string `json:"sub"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.StandardClaims
}

type AuthService struct {
    BaseURL string
}

type UserContext struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Role  string `json:"role"`
}

func NewAuthService(baseURL string) *AuthService {
    return &AuthService{BaseURL: baseURL}
}

func (a *AuthService) ValidateToken(tokenString string) (*UserContext, error) {
    secretKey := os.Getenv("JWT_SECRET_KEY")
    if secretKey == "" {
        secretKey = "streamflow_secret_key_2024"
    }

    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(secretKey), nil
    })

    if err != nil || !token.Valid {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok {
        return nil, jwt.ErrInvalidKey
    }

    return &UserContext{
        ID:    claims.UserID,
        Email: claims.Email,
        Role:  claims.Role,
    }, nil
}

func authMiddleware(authService *AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        
        // Endpoints públicos
        publicEndpoints := []string{
            "POST /usuarios",
            "POST /auth/login",
            "GET /health",
            "GET /videos",
            "GET /videos/",
            "GET /comedia",
        }
        
        method := c.Request.Method
        path := c.Request.URL.Path
        endpoint := method + " " + path
        
        for _, public := range publicEndpoints {
            if strings.HasPrefix(endpoint, public) {
                c.Next()
                return
            }
        }
        
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token de autorización requerido"})
            c.Abort()
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        user, err := authService.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
            c.Abort()
            return
        }
        
        c.Set("user", user)
        c.Next()
    }
}

func proxyToAuthService(authServiceURL string) gin.HandlerFunc {
    return func(c *gin.Context) {
        targetURL := authServiceURL + c.Request.URL.Path
        
        var body []byte
        if c.Request.Body != nil {
            body, _ = io.ReadAll(c.Request.Body)
        }
        
        req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewBuffer(body))
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando petición"})
            return
        }
        
        for key, values := range c.Request.Header {
            for _, value := range values {
                req.Header.Add(key, value)
            }
        }
        
        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de autenticación"})
            return
        }
        defer resp.Body.Close()
        
        respBody, _ := io.ReadAll(resp.Body)
        
        for key, values := range resp.Header {
            for _, value := range values {
                c.Header(key, value)
            }
        }
        
        c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
    }
}

// gRPC client para usuarios
func getUsersClient() (pb.UserServiceClient, *grpc.ClientConn, error) {
    usersServiceURL := os.Getenv("USERS_SERVICE_URL")
    if usersServiceURL == "" {
        usersServiceURL = "localhost:50051"
    }
    
    conn, err := grpc.Dial(usersServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, nil, err
    }
    
    client := pb.NewUserServiceClient(conn)
    return client, conn, nil
}

func createUser(c *gin.Context) {
    client, conn, err := getUsersClient()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de usuarios: " + err.Error()})
        return
    }
    defer conn.Close()
    
    var requestBody map[string]interface{}
    if err := c.ShouldBindJSON(&requestBody); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
        return
    }
    
    // Convertir del JSON a la estructura gRPC
    // Mapear "name" a first_name si no se proporciona first_name
    firstName := getString(requestBody, "first_name")
    lastName := getString(requestBody, "last_name")
    
    // Si viene "name" pero no first_name/last_name, dividir el nombre
    if firstName == "" && lastName == "" {
        fullName := getString(requestBody, "name")
        if fullName != "" {
            parts := strings.Fields(fullName)
            if len(parts) > 0 {
                firstName = parts[0]
            }
            if len(parts) > 1 {
                lastName = strings.Join(parts[1:], " ")
            }
        }
    }
    
    request := &pb.CreateUserRequest{
        Email:           getString(requestBody, "email"),
        Password:        getString(requestBody, "password"),
        ConfirmPassword: getString(requestBody, "confirm_password"),
        FirstName:       firstName,
        LastName:        lastName,
        Role:            getString(requestBody, "role"),
    }
    
    if request.Role == "" {
        request.Role = "cliente"
    }
    
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    response, err := client.CreateUser(ctx, request)
    if err != nil {
        // Map gRPC errors to appropriate HTTP status codes
        statusCode := mapGRPCErrorToHTTP(err)
        c.JSON(statusCode, gin.H{"error": "Error creando usuario: " + err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, response)
}

func getUser(c *gin.Context) {
    client, conn, err := getUsersClient()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de usuarios: " + err.Error()})
        return
    }
    defer conn.Close()
    
    userID := c.Param("id")
    id, err := strconv.ParseInt(userID, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
        return
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    response, err := client.GetUser(ctx, &pb.GetUserRequest{Id: int32(id)})
    if err != nil {
        statusCode := mapGRPCErrorToHTTP(err)
        c.JSON(statusCode, gin.H{"error": "Usuario no encontrado: " + err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}

func updateUser(c *gin.Context) {
    client, conn, err := getUsersClient()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de usuarios: " + err.Error()})
        return
    }
    defer conn.Close()
    
    userID := c.Param("id")
    id, err := strconv.ParseInt(userID, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
        return
    }
    
    var requestBody map[string]interface{}
    if err := c.ShouldBindJSON(&requestBody); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
        return
    }
    
    // Mapear campos como en CreateUser
    firstName := getString(requestBody, "first_name")
    lastName := getString(requestBody, "last_name")
    
    if firstName == "" && lastName == "" {
        fullName := getString(requestBody, "name")
        if fullName != "" {
            parts := strings.Fields(fullName)
            if len(parts) > 0 {
                firstName = parts[0]
            }
            if len(parts) > 1 {
                lastName = strings.Join(parts[1:], " ")
            }
        }
    }
    
    request := &pb.UpdateUserRequest{
        Id:        int32(id),
        Email:     getString(requestBody, "email"),
        FirstName: firstName,
        LastName:  lastName,
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    response, err := client.UpdateUser(ctx, request)
    if err != nil {
        statusCode := mapGRPCErrorToHTTP(err)
        c.JSON(statusCode, gin.H{"error": "Error actualizando usuario: " + err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}

func deleteUser(c *gin.Context) {
    client, conn, err := getUsersClient()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de usuarios: " + err.Error()})
        return
    }
    defer conn.Close()
    
    userID := c.Param("id")
    id, err := strconv.ParseInt(userID, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
        return
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    _, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: int32(id)})
    if err != nil {
        statusCode := mapGRPCErrorToHTTP(err)
        c.JSON(statusCode, gin.H{"error": "Error eliminando usuario: " + err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Usuario eliminado exitosamente"})
}

func listUsers(c *gin.Context) {
    client, conn, err := getUsersClient()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de usuarios: " + err.Error()})
        return
    }
    defer conn.Close()
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    response, err := client.ListUsers(ctx, &pb.ListUsersRequest{})
    if err != nil {
        statusCode := mapGRPCErrorToHTTP(err)
        c.JSON(statusCode, gin.H{"error": "Error listando usuarios: " + err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}

// Función auxiliar para extraer strings del JSON
func getString(data map[string]interface{}, key string) string {
    if val, ok := data[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    return ""
}

// mapGRPCErrorToHTTP maps gRPC error codes to HTTP status codes
func mapGRPCErrorToHTTP(err error) int {
    st, ok := status.FromError(err)
    if !ok {
        return http.StatusInternalServerError
    }
    
    switch st.Code() {
    case codes.OK:
        return http.StatusOK
    case codes.InvalidArgument:
        return http.StatusBadRequest
    case codes.NotFound:
        return http.StatusNotFound
    case codes.AlreadyExists:
        return http.StatusConflict
    case codes.PermissionDenied:
        return http.StatusForbidden
    case codes.Unauthenticated:
        return http.StatusUnauthorized
    case codes.ResourceExhausted:
        return http.StatusTooManyRequests
    case codes.FailedPrecondition:
        return http.StatusPreconditionFailed
    case codes.Aborted:
        return http.StatusConflict
    case codes.OutOfRange:
        return http.StatusBadRequest
    case codes.Unimplemented:
        return http.StatusNotImplemented
    case codes.Internal:
        return http.StatusInternalServerError
    case codes.Unavailable:
        return http.StatusServiceUnavailable
    case codes.DataLoss:
        return http.StatusInternalServerError
    default:
        return http.StatusInternalServerError
    }
}

// gRPC client para videos
func getVideosClient() (pb.VideoServiceClient, *grpc.ClientConn, error) {
    videosServiceURL := os.Getenv("VIDEOS_SERVICE_URL")
    if videosServiceURL == "" {
        videosServiceURL = "localhost:50053"
    }
    
    conn, err := grpc.Dial(videosServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, nil, err
    }
    
    client := pb.NewVideoServiceClient(conn)
    return client, conn, nil
}

func listVideos(c *gin.Context) {
    client, conn, err := getVideosClient()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de videos: " + err.Error()})
        return
    }
    defer conn.Close()
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    response, err := client.ListVideos(ctx, &pb.ListVideosRequest{})
    if err != nil {
        statusCode := mapGRPCErrorToHTTP(err)
        c.JSON(statusCode, gin.H{"error": "Error listando videos: " + err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}

func getVideo(c *gin.Context) {
    client, conn, err := getVideosClient()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando al servicio de videos: " + err.Error()})
        return
    }
    defer conn.Close()
    
    videoID := c.Param("id")
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    response, err := client.GetVideo(ctx, &pb.GetVideoRequest{Id: videoID})
    if err != nil {
        statusCode := mapGRPCErrorToHTTP(err)
        c.JSON(statusCode, gin.H{"error": "Error obteniendo video: " + err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}

func handleVideos(c *gin.Context) {
    // Fallback for unimplemented video endpoints
    c.JSON(http.StatusOK, gin.H{
        "message": "Videos service - Endpoint específico no implementado",
        "method": c.Request.Method,
        "path": c.Request.URL.Path,
    })
}

func handleBilling(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Billing service - Implementación pendiente",
        "method": c.Request.Method,
        "path": c.Request.URL.Path,
    })
}

func handleMonitoring(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Monitoring service - Implementación pendiente",
        "method": c.Request.Method,
        "path": c.Request.URL.Path,
    })
}

func handlePlaylists(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Playlists service - Implementación pendiente",
        "method": c.Request.Method,
        "path": c.Request.URL.Path,
    })
}

func handleSocial(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Social service - Implementación pendiente",
        "method": c.Request.Method,
        "path": c.Request.URL.Path,
    })
}

func main() {
    authServiceURL := os.Getenv("AUTH_SERVICE_URL")
    if authServiceURL == "" {
        authServiceURL = "http://localhost:8001"
    }

    authService := NewAuthService(authServiceURL)
    
    router := gin.Default()
    
    // Middleware de autenticación
    router.Use(authMiddleware(authService))
    
    // Middleware de logging
    router.Use(gin.Logger())
    router.Use(gin.Recovery())
    
    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "healthy",
            "service": "api-gateway",
        })
    })
    
    // Endpoint cómico para Nginx
    router.GET("/comedia", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "¿Por qué los programadores prefieren el modo oscuro? Porque la luz atrae bugs! 🐛💡",
            "service": "nginx-comedy-endpoint",
        })
    })
    
    // Rutas de autenticación (proxy directo)
    authGroup := router.Group("/auth")
    {
        authGroup.Any("/*path", proxyToAuthService(authServiceURL))
    }
    
    // Rutas de usuarios
    userGroup := router.Group("/usuarios")
    {
        userGroup.POST("", createUser)
        userGroup.GET("/:id", getUser)
        userGroup.PATCH("/:id", updateUser)
        userGroup.DELETE("/:id", deleteUser)
        userGroup.GET("", listUsers)
    }
    
    // Rutas de facturas
    billGroup := router.Group("/facturas")
    {
        billGroup.POST("", handleBilling)
        billGroup.GET("/:id", handleBilling)
        billGroup.PATCH("/:id", handleBilling)
        billGroup.DELETE("/:id", handleBilling)
        billGroup.GET("", handleBilling)
    }
    
    // Rutas de videos
    videoGroup := router.Group("/videos")
    {
        videoGroup.POST("", handleVideos)    // POST /videos - Not implemented yet
        videoGroup.GET("/:id", getVideo)    // GET /videos/:id
        videoGroup.PATCH("/:id", handleVideos) // PATCH /videos/:id - Not implemented yet
        videoGroup.DELETE("/:id", handleVideos) // DELETE /videos/:id - Not implemented yet
        videoGroup.GET("", listVideos)     // GET /videos
    }
    
    // Rutas de monitoreo
    monitoringGroup := router.Group("/monitoreo")
    {
        monitoringGroup.GET("/acciones", handleMonitoring)
        monitoringGroup.GET("/errores", handleMonitoring)
    }
    
    // Rutas de listas de reproducción
    playlistGroup := router.Group("/listas-reproduccion")
    {
        playlistGroup.POST("", handlePlaylists)
        playlistGroup.POST("/:id/videos", handlePlaylists)
        playlistGroup.GET("", handlePlaylists)
        playlistGroup.GET("/:id/videos", handlePlaylists)
        playlistGroup.DELETE("/:id/videos", handlePlaylists)
        playlistGroup.DELETE("/:id", handlePlaylists)
    }
    
    // Rutas de interacciones sociales
    socialGroup := router.Group("/interacciones")
    {
        socialGroup.POST("/:id/likes", handleSocial)
        socialGroup.POST("/:id/comentarios", handleSocial)
        socialGroup.GET("/:id", handleSocial)
    }
 
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("API Gateway iniciado en puerto %s", port)
    router.Run(":" + port)
}