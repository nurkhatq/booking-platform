package handlers

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "booking-platform/shared/i18n"
    "booking-platform/shared/models"
    pb "booking-platform/user-service/proto"
)

type Handler struct {
    grpc map[string]*grpc.ClientConn
}

// NewHandler creates a new handler with gRPC connections
func NewHandler(grpcClients map[string]*grpc.ClientConn) *Handler {
    if grpcClients == nil {
        panic("gRPC clients map cannot be nil")
    }
    
    // Verify required services are connected
    requiredServices := []string{"user", "booking", "notification", "payment", "admin"}
    for _, service := range requiredServices {
        if grpcClients[service] == nil {
            panic("gRPC client for " + service + " service is not initialized")
        }
    }

    return &Handler{
        grpc: grpcClients,
    }
}

func (h *Handler) Login(c *gin.Context) {
    var req models.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }

    language := c.GetString("language")

    // Call user service via gRPC to authenticate
    userClient := pb.NewUserServiceClient(h.grpc["user"])

    // Create context with timeout
    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()

    grpcReq := &pb.LoginRequest{
        Email:    req.Email,
        Password: req.Password,
    }

    grpcResp, err := userClient.Login(ctx, grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.Unauthenticated:
                c.JSON(http.StatusUnauthorized, gin.H{
                    "error": "invalid_credentials",
                    "message": i18n.T(language, "auth.login.invalid_credentials"),
                })
                return
            case codes.NotFound:
                c.JSON(http.StatusUnauthorized, gin.H{
                    "error": "user_not_found", 
                    "message": i18n.T(language, "auth.login.user_not_found"),
                })
                return
            case codes.DeadlineExceeded:
                c.JSON(http.StatusRequestTimeout, gin.H{
                    "error": "timeout",
                    "message": i18n.T(language, "error.timeout"),
                })
                return
            case codes.Unavailable:
                c.JSON(http.StatusServiceUnavailable, gin.H{
                    "error": "service_unavailable",
                    "message": i18n.T(language, "error.service_unavailable"),
                })
                return
            default:
                // Log the actual error for debugging
                c.Header("X-Debug-Error", st.Message())
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        
        // Non-gRPC error - log for debugging
        c.Header("X-Debug-Error", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }

    // Successful login
    c.JSON(http.StatusOK, gin.H{
        "token":         grpcResp.Token,
        "refresh_token": grpcResp.RefreshToken,
        "user":          grpcResp.User,
        "message":       i18n.T(language, "auth.login.success"),
    })
}

func (h *Handler) AdminLogin(c *gin.Context) {
    var req models.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }

    language := c.GetString("language")

    // Call user service via gRPC to authenticate
    userClient := pb.NewUserServiceClient(h.grpc["user"])

    // Create context with timeout
    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()

    grpcReq := &pb.LoginRequest{
        Email:    req.Email,
        Password: req.Password,
    }

    grpcResp, err := userClient.Login(ctx, grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.Unauthenticated:
                c.JSON(http.StatusUnauthorized, gin.H{
                    "error": "invalid_credentials",
                    "message": i18n.T(language, "auth.login.invalid_credentials"),
                })
                return
            case codes.NotFound:
                c.JSON(http.StatusUnauthorized, gin.H{
                    "error": "user_not_found",
                    "message": i18n.T(language, "auth.login.user_not_found"),
                })
                return
            case codes.DeadlineExceeded:
                c.JSON(http.StatusRequestTimeout, gin.H{
                    "error": "timeout",
                    "message": i18n.T(language, "error.timeout"),
                })
                return
            case codes.Unavailable:
                c.JSON(http.StatusServiceUnavailable, gin.H{
                    "error": "service_unavailable",
                    "message": i18n.T(language, "error.service_unavailable"),
                })
                return
            default:
                // Log the actual error for debugging
                c.Header("X-Debug-Error", st.Message())
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        
        // Non-gRPC error - log for debugging
        c.Header("X-Debug-Error", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }

    // Check if user has SUPER_ADMIN role
    if grpcResp.User.Role != "SUPER_ADMIN" {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "insufficient_permissions",
            "message": "Admin access required",
        })
        return
    }

    // Successful admin login
    c.JSON(http.StatusOK, gin.H{
        "token":         grpcResp.Token,
        "refresh_token": grpcResp.RefreshToken,
        "user":          grpcResp.User,
        "message":       i18n.T(language, "auth.login.success"),
    })
}