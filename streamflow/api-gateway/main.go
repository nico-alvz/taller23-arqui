package main

import (
    "log"
    "net/http"
    "os"
    "strings"
    "bytes"
    "io"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
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

func handleUsers(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Users service - Implementación pendiente",
        "method": c.Request.Method,
        "path": c.Request.URL.Path,
    })
}

func handleVideos(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Videos service - Implementación pendiente",
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
        userGroup.POST("", handleUsers)
        userGroup.GET("/:id", handleUsers)
        userGroup.PATCH("/:id", handleUsers)
        userGroup.DELETE("/:id", handleUsers)
        userGroup.GET("", handleUsers)
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
        videoGroup.POST("", handleVideos)
        videoGroup.GET("/:id", handleVideos)
        videoGroup.PATCH("/:id", handleVideos)
        videoGroup.DELETE("/:id", handleVideos)
        videoGroup.GET("", handleVideos)
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