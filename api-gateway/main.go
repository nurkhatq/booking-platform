package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/keepalive"

    "booking-platform/api-gateway/handlers"
    "booking-platform/api-gateway/middleware"
    "booking-platform/api-gateway/routes"
    "booking-platform/shared/config"
    "booking-platform/shared/database"
    "booking-platform/shared/cache"
)

func main() {
    // Load configuration
    cfg := config.Load()

    // Initialize database
    err := database.Initialize(cfg)
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    log.Println("Database connection established successfully")

    // Initialize Redis
    err = cache.Initialize(cfg)
    if err != nil {
        log.Fatal("Failed to initialize Redis:", err)
    }
    log.Println("Redis connection established successfully")

    // Initialize gRPC clients with retry and keepalive
    grpcClients := initializeGRPCClients(cfg)
    defer closeGRPCClients(grpcClients)

    // Initialize handlers with gRPC clients
    handler := handlers.NewHandler(grpcClients)

    // Setup Gin router
    if os.Getenv("GIN_MODE") == "release" {
        gin.SetMode(gin.ReleaseMode)
    }

    r := gin.Default()

    // Add middleware
    r.Use(middleware.CORS())
    r.Use(middleware.Logger())
    r.Use(middleware.ErrorHandler())
    r.Use(middleware.RequestID())
    r.Use(middleware.RateLimit())
    r.Use(middleware.I18n())

    // Setup routes
    routes.SetupRoutes(r, handler)

    // Health check endpoint
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "service": "api-gateway",
            "status":  "healthy",
        })
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("API Gateway starting on port :%s", port)
    if err := r.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}

func initializeGRPCClients(cfg *config.Config) map[string]*grpc.ClientConn {
    grpcClients := make(map[string]*grpc.ClientConn)

    // gRPC connection options with keepalive and retry
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             3 * time.Second,
            PermitWithoutStream: true,
        }),
        grpc.WithDefaultCallOptions(
            grpc.WaitForReady(true),
        ),
    }

    // User Service connection
    userServiceAddr := fmt.Sprintf("%s:%s", 
        getEnvOrDefault("USER_SERVICE_HOST", "user-service"),
        getEnvOrDefault("USER_SERVICE_GRPC_PORT", "50051"),
    )
    
    log.Printf("Connecting to User Service at %s", userServiceAddr)
    userConn, err := grpc.Dial(userServiceAddr, opts...)
    if err != nil {
        log.Fatal("Failed to connect to User Service:", err)
    }
    grpcClients["user"] = userConn
    log.Println("User Service gRPC connection established")

    // Booking Service connection
    bookingServiceAddr := fmt.Sprintf("%s:%s",
        getEnvOrDefault("BOOKING_SERVICE_HOST", "booking-service"),
        getEnvOrDefault("BOOKING_SERVICE_GRPC_PORT", "50052"),
    )
    
    log.Printf("Connecting to Booking Service at %s", bookingServiceAddr)
    bookingConn, err := grpc.Dial(bookingServiceAddr, opts...)
    if err != nil {
        log.Fatal("Failed to connect to Booking Service:", err)
    }
    grpcClients["booking"] = bookingConn
    log.Println("Booking Service gRPC connection established")

    // Notification Service connection
    notificationServiceAddr := fmt.Sprintf("%s:%s",
        getEnvOrDefault("NOTIFICATION_SERVICE_HOST", "notification-service"),
        getEnvOrDefault("NOTIFICATION_SERVICE_GRPC_PORT", "50053"),
    )
    
    log.Printf("Connecting to Notification Service at %s", notificationServiceAddr)
    notificationConn, err := grpc.Dial(notificationServiceAddr, opts...)
    if err != nil {
        log.Fatal("Failed to connect to Notification Service:", err)
    }
    grpcClients["notification"] = notificationConn
    log.Println("Notification Service gRPC connection established")

    // Payment Service connection
    paymentServiceAddr := fmt.Sprintf("%s:%s",
        getEnvOrDefault("PAYMENT_SERVICE_HOST", "payment-service"),
        getEnvOrDefault("PAYMENT_SERVICE_GRPC_PORT", "50054"),
    )
    
    log.Printf("Connecting to Payment Service at %s", paymentServiceAddr)
    paymentConn, err := grpc.Dial(paymentServiceAddr, opts...)
    if err != nil {
        log.Fatal("Failed to connect to Payment Service:", err)
    }
    grpcClients["payment"] = paymentConn
    log.Println("Payment Service gRPC connection established")

    // Admin Service connection
    adminServiceAddr := fmt.Sprintf("%s:%s",
        getEnvOrDefault("ADMIN_SERVICE_HOST", "admin-service"),
        getEnvOrDefault("ADMIN_SERVICE_GRPC_PORT", "50055"),
    )
    
    log.Printf("Connecting to Admin Service at %s", adminServiceAddr)
    adminConn, err := grpc.Dial(adminServiceAddr, opts...)
    if err != nil {
        log.Fatal("Failed to connect to Admin Service:", err)
    }
    grpcClients["admin"] = adminConn
    log.Println("Admin Service gRPC connection established")

    log.Println("All gRPC connections established successfully")
    return grpcClients
}

func closeGRPCClients(clients map[string]*grpc.ClientConn) {
    for name, conn := range clients {
        if err := conn.Close(); err != nil {
            log.Printf("Error closing %s gRPC connection: %v", name, err)
        }
    }
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}