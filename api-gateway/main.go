package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "google.golang.org/grpc"
    
    "booking-platform/api-gateway/handlers"
    "booking-platform/api-gateway/middleware"
    "booking-platform/api-gateway/routes"
    "booking-platform/shared/config"
    "booking-platform/shared/database"
    "booking-platform/shared/cache"
    "booking-platform/shared/auth"
    "booking-platform/shared/i18n"
)

func main() {
    // Load configuration
    cfg := config.Load()
    
    // Initialize dependencies
    if err := database.Initialize(cfg); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer database.Close()
    
    if err := cache.Initialize(cfg); err != nil {
        log.Fatalf("Failed to initialize cache: %v", err)
    }
    defer cache.Close()
    
    auth.Initialize(cfg)
    
    if err := i18n.Initialize(cfg); err != nil {
        log.Fatalf("Failed to initialize i18n: %v", err)
    }
    
    // Initialize gRPC connections
    grpcConnections := initializeGRPCConnections(cfg)
    defer closeGRPCConnections(grpcConnections)
    
    // Initialize handlers
    h := handlers.NewHandler(cfg, grpcConnections)
    
if cfg.Environment == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    r := gin.Default()
    
    // Configure CORS
    corsConfig := cors.DefaultConfig()
    corsConfig.AllowOrigins = []string{
        "https://*.jazyl.tech",
        "https://jazyl.tech",
        "http://localhost:3000", // For development
    }
    corsConfig.AllowCredentials = true
    corsConfig.AllowHeaders = []string{
        "Origin", "Content-Length", "Content-Type", "Authorization",
        "X-Requested-With", "Accept", "Accept-Language", "X-Subdomain",
    }
    r.Use(cors.New(corsConfig))
    
    // Add middleware
    r.Use(middleware.RequestLogger())
    r.Use(middleware.RateLimit(cfg))
    r.Use(middleware.SubdomainExtractor())
    r.Use(middleware.LanguageDetector(cfg))
    
    // Health check endpoint
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "healthy",
            "service": "api-gateway",
        })
    })
    
    // Setup routes
    routes.SetupRoutes(r, h, cfg)
    
    // Start server
    port := fmt.Sprintf(":%d", cfg.Services.APIGateway)
    log.Printf("API Gateway starting on port %s", port)
    
    // Graceful shutdown
    go func() {
        if err := r.Run(port); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Failed to start server: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("API Gateway shutting down...")
}

func initializeGRPCConnections(cfg *config.Config) map[string]*grpc.ClientConn {
    connections := make(map[string]*grpc.ClientConn)
    
    // User Service
    userConn, err := grpc.Dial(
        fmt.Sprintf("user-service:%d", cfg.Services.UserGRPC),
        grpc.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("Failed to connect to user service: %v", err)
    }
    connections["user"] = userConn
    
    // Booking Service
    bookingConn, err := grpc.Dial(
        fmt.Sprintf("booking-service:%d", cfg.Services.BookingGRPC),
        grpc.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("Failed to connect to booking service: %v", err)
    }
    connections["booking"] = bookingConn
    
    // Notification Service
    notificationConn, err := grpc.Dial(
        fmt.Sprintf("notification-service:%d", cfg.Services.NotificationGRPC),
        grpc.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("Failed to connect to notification service: %v", err)
    }
    connections["notification"] = notificationConn
    
    // Payment Service
    paymentConn, err := grpc.Dial(
        fmt.Sprintf("payment-service:%d", cfg.Services.PaymentGRPC),
        grpc.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("Failed to connect to payment service: %v", err)
    }
    connections["payment"] = paymentConn
    
    // Admin Service
    adminConn, err := grpc.Dial(
        fmt.Sprintf("admin-service:%d", cfg.Services.AdminGRPC),
        grpc.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("Failed to connect to admin service: %v", err)
    }
    connections["admin"] = adminConn
    
    return connections
}

func closeGRPCConnections(connections map[string]*grpc.ClientConn) {
    for name, conn := range connections {
        if err := conn.Close(); err != nil {
            log.Printf("Error closing %s connection: %v", name, err)
        }
    }
}
