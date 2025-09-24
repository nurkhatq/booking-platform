package main

import (
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    
    "booking-platform/shared/config"
    "booking-platform/shared/database"
    "booking-platform/shared/cache"
    "booking-platform/shared/i18n"
    "booking-platform/notification-service/services"
    "booking-platform/notification-service/workers"
    pb "booking-platform/notification-service/proto"
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
    
    if err := i18n.Initialize(cfg); err != nil {
        log.Fatalf("Failed to initialize i18n: %v", err)
    }
    
    // Initialize services
    notificationService := services.NewNotificationService(cfg)
    jobManager := workers.NewJobManager(cfg)
    
    // Start background workers
    go jobManager.StartWorkers()
    defer jobManager.Stop()
    
    // Start gRPC server
    go func() {
        lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Services.NotificationGRPC))
        if err != nil {
            log.Fatalf("Failed to listen: %v", err)
        }
        
        s := grpc.NewServer()
        pb.RegisterNotificationServiceServer(s, notificationService)
        
        log.Printf("Notification Service gRPC server listening on port %d", cfg.Services.NotificationGRPC)
        if err := s.Serve(lis); err != nil {
            log.Fatalf("Failed to serve gRPC: %v", err)
        }
    }()
    
    // Start HTTP server for health checks
    if cfg.Environment != "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    r := gin.Default()
    
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "healthy",
            "service": "notification-service",
        })
    })
    
    go func() {
        port := fmt.Sprintf(":%d", cfg.Services.NotificationHTTP)
        log.Printf("Notification Service HTTP server listening on port %s", port)
        if err := r.Run(port); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Failed to start HTTP server: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Notification Service shutting down...")
}
