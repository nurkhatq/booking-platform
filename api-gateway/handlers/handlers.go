package handlers

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    
    "booking-platform/shared/config"
    "booking-platform/shared/models"
    "booking-platform/shared/i18n"
    "booking-platform/shared/auth"
    pb "booking-platform/user-service/proto"
    bookingPb "booking-platform/booking-service/proto"
    adminPb "booking-platform/admin-service/proto"
)

type Handler struct {
    config *config.Config
    grpc   map[string]*grpc.ClientConn
}

func NewHandler(cfg *config.Config, grpcConnections map[string]*grpc.ClientConn) *Handler {
    return &Handler{
        config: cfg,
        grpc:   grpcConnections,
    }
}

// Public API Handlers
func (h *Handler) GetBusinessInfo(c *gin.Context) {
    _ = c.Param("subdomain") // subdomain will be used in future implementation
    language := c.GetString("language")
    
    // Call user service via gRPC to get business info
    // Implementation would use gRPC client
    
    c.JSON(http.StatusOK, gin.H{
        "business_name": "Sample Business",
        "business_type": "salon",
        "message": i18n.T(language, "business.info.success"),
    })
}

func (h *Handler) RegisterBusiness(c *gin.Context) {
    var req models.TenantRegistrationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    language := c.GetString("language")
    
    // Call user service via gRPC to register business
    // Implementation would use gRPC client
    
    c.JSON(http.StatusCreated, gin.H{
        "message": i18n.T(language, "tenant.registration.pending"),
        "tenant_id": uuid.New().String(),
    })
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
    
    grpcReq := &pb.LoginRequest{
        Email:    req.Email,
        Password: req.Password,
    }
    
    grpcResp, err := userClient.Login(c.Request.Context(), grpcReq)
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
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "token": grpcResp.Token,
        "refresh_token": grpcResp.RefreshToken,
        "message": i18n.T(language, "auth.login.success"),
    })
}

func (h *Handler) VerifyClient(c *gin.Context) {
    var req models.ClientVerificationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    language := c.GetString("language")
    
    // Call user service via gRPC to create client session and send verification
    // Implementation would use gRPC client
    
    c.JSON(http.StatusOK, models.ClientVerificationResponse{
        SessionID: uuid.New().String(),
        Message:   i18n.T(language, "client.verification.sent"),
    })
}

func (h *Handler) VerifyClientCode(c *gin.Context) {
    var req models.VerifyCodeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    language := c.GetString("language")
    
    // Call user service via gRPC to verify code and create token
    // Implementation would use gRPC client
    
    c.JSON(http.StatusOK, models.VerifyCodeResponse{
        Token:   "client_jwt_token",
        Message: i18n.T(language, "client.verification.success"),
    })
}

func (h *Handler) CreatePublicBooking(c *gin.Context) {
    var req models.BookingRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    _ = c.GetString("subdomain") // subdomain will be used in future implementation
    language := c.GetString("language")
    
    // First verify client
    _ = models.ClientVerificationRequest{ // clientSession will be used in future implementation
        Email: req.ClientEmail,
        Phone: req.ClientPhone,
        Name:  req.ClientName,
    }
    
    // Call user service to create/verify client session
    // Call booking service to create booking
    // Implementation would use gRPC clients
    
    c.JSON(http.StatusCreated, gin.H{
        "booking_id": uuid.New().String(),
        "confirmation_code": "ABC123",
        "message": i18n.T(language, "booking.created"),
    })
}

// Authenticated Handlers
func (h *Handler) GetProfile(c *gin.Context) {
    userID := c.GetString("user_id")
    language := c.GetString("language")
    
    // Call user service via gRPC to get user profile
    // Implementation would use gRPC client
    
    c.JSON(http.StatusOK, gin.H{
        "user_id": userID,
        "message": i18n.T(language, "profile.retrieved"),
    })
}

func (h *Handler) GetDashboard(c *gin.Context) {
    _ = c.GetString("user_id") // userID will be used in future implementation
    _ = c.GetString("tenant_id") // tenantID will be used in future implementation
    role := c.GetString("user_role") // role will be used in future implementation
    language := c.GetString("language")
    
    // Call appropriate services based on role
    // Owner: All locations stats, booking stats, revenue
    // Manager: Location-specific stats
    // Master: Personal booking stats, schedule
    
    c.JSON(http.StatusOK, gin.H{
        "role": role,
        "data": gin.H{},
        "message": i18n.T(language, "dashboard.loaded"),
    })
}

func (h *Handler) GetBookings(c *gin.Context) {
    _ = c.GetString("user_id") // userID will be used in future implementation
    _ = c.GetString("user_role") // role will be used in future implementation
    language := c.GetString("language")
    
    // Parse query parameters
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    _ = c.Query("status") // status will be used in future implementation
    _ = c.Query("date_from") // dateFrom will be used in future implementation
    _ = c.Query("date_to") // dateTo will be used in future implementation
    
    // Call booking service via gRPC
    // Filter based on user role and permissions
    
    c.JSON(http.StatusOK, gin.H{
        "bookings": []models.Booking{},
        "pagination": gin.H{
            "page":  page,
            "limit": limit,
            "total": 0,
        },
        "message": i18n.T(language, "bookings.retrieved"),
    })
}

func (h *Handler) CreateBooking(c *gin.Context) {
    var req models.BookingRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    _ = c.GetString("user_id") // userID will be used in future implementation
    role := c.GetString("user_role")
    language := c.GetString("language")
    
    // Validate permissions based on role
    if role == "MASTER" {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "insufficient_permissions",
            "message": i18n.T(language, "error.forbidden"),
        })
        return
    }
    
    // Call booking service via gRPC
    // Implementation would use gRPC client
    
    c.JSON(http.StatusCreated, gin.H{
        "booking_id": uuid.New().String(),
        "message": i18n.T(language, "booking.created"),
    })
}

// Admin Handlers
func (h *Handler) GetPendingTenants(c *gin.Context) {
    language := c.GetString("language")
    
    // Parse query parameters
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    // Call admin service via gRPC
    // Implementation would use gRPC client
    
    c.JSON(http.StatusOK, gin.H{
        "tenants": []models.Tenant{},
        "pagination": gin.H{
            "page":  page,
            "limit": limit,
            "total": 0,
        },
        "message": i18n.T(language, "tenants.retrieved"),
    })
}

func (h *Handler) ApproveTenant(c *gin.Context) {
    tenantID := c.Param("id")
    language := c.GetString("language")
    
    // Validate UUID
    if _, err := uuid.Parse(tenantID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid_id",
            "message": "Invalid tenant ID",
        })
        return
    }
    
    // Parse trial days from request
    var req struct {
        TrialDays int    `json:"trial_days"`
        Notes     string `json:"notes"`
    }
    
    if err := c.ShouldBindJSON(&req); err == nil && req.TrialDays > 0 {
        // Use custom trial days
    } else {
        req.TrialDays = h.config.Business.DefaultTrialDays
    }
    
    // Call admin service via gRPC to approve tenant
    // Implementation would use gRPC client
    
    c.JSON(http.StatusOK, gin.H{
        "message": i18n.T(language, "tenant.approved"),
        "trial_days": req.TrialDays,
    })
}

func (h *Handler) RejectTenant(c *gin.Context) {
    tenantID := c.Param("id")
    language := c.GetString("language")
    
    // Validate UUID
    if _, err := uuid.Parse(tenantID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid_id",
            "message": "Invalid tenant ID",
        })
        return
    }
    
    var req struct {
        Reason string `json:"reason" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    // Call admin service via gRPC to reject tenant
    // Implementation would use gRPC client
    
    c.JSON(http.StatusOK, gin.H{
        "message": i18n.T(language, "tenant.rejected"),
    })
}

// Placeholder implementations for other handlers
func (h *Handler) Logout(c *gin.Context) {
    // Get token from Authorization header
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "missing_token",
            "message": "Authorization header is required",
        })
        return
    }
    
    // Extract token (assuming "Bearer <token>" format)
    token := authHeader
    if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
        token = authHeader[7:]
    }
    
    language := c.GetString("language")
    
    // Call user service via gRPC to logout
    userClient := pb.NewUserServiceClient(h.grpc["user"])
    
    grpcReq := &pb.LogoutRequest{
        Token: token,
    }
    
    _, err := userClient.Logout(c.Request.Context(), grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.Unauthenticated:
                c.JSON(http.StatusUnauthorized, gin.H{
                    "error": "invalid_token",
                    "message": i18n.T(language, "auth.logout.invalid_token"),
                })
                return
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": i18n.T(language, "auth.logout.success"),
    })
}
func (h *Handler) RefreshToken(c *gin.Context) {
    var req models.RefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    language := c.GetString("language")
    
    // Call user service via gRPC to refresh token
    userClient := pb.NewUserServiceClient(h.grpc["user"])
    
    grpcReq := &pb.RefreshTokenRequest{
        RefreshToken: req.RefreshToken,
    }
    
    grpcResp, err := userClient.RefreshToken(c.Request.Context(), grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.Unauthenticated:
                c.JSON(http.StatusUnauthorized, gin.H{
                    "error": "invalid_refresh_token",
                    "message": i18n.T(language, "auth.refresh.invalid_token"),
                })
                return
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "token": grpcResp.Token,
        "refresh_token": grpcResp.RefreshToken,
        "message": i18n.T(language, "auth.refresh.success"),
    })
}
func (h *Handler) GetBusinessLocations(c *gin.Context)        { h.notImplemented(c) }
func (h *Handler) GetBusinessServices(c *gin.Context)         { h.notImplemented(c) }
func (h *Handler) GetBusinessMasters(c *gin.Context)          { h.notImplemented(c) }
func (h *Handler) CheckAvailability(c *gin.Context) {
    var req struct {
        TenantID   string `json:"tenant_id" binding:"required"`
        LocationID string `json:"location_id" binding:"required"`
        MasterID   string `json:"master_id" binding:"required"`
        ServiceID  string `json:"service_id" binding:"required"`
        Date       string `json:"date" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    language := c.GetString("language")
    
    // Call booking service via gRPC
    bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])
    
    grpcReq := &bookingPb.CheckAvailabilityRequest{
        TenantId:   req.TenantID,
        LocationId: req.LocationID,
        MasterId:   req.MasterID,
        ServiceId:  req.ServiceID,
        Date:       req.Date,
    }
    
    grpcResp, err := bookingClient.CheckAvailability(c.Request.Context(), grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.InvalidArgument:
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": "invalid_argument",
                    "message": st.Message(),
                })
                return
            case codes.NotFound:
                c.JSON(http.StatusNotFound, gin.H{
                    "error": "not_found",
                    "message": st.Message(),
                })
                return
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "available_slots": grpcResp.AvailableSlots,
    })
}
func (h *Handler) GetClientBookings(c *gin.Context)           { h.notImplemented(c) }
func (h *Handler) GetClientBooking(c *gin.Context)            { h.notImplemented(c) }
func (h *Handler) CancelClientBooking(c *gin.Context)         { h.notImplemented(c) }
func (h *Handler) UpdateClientProfile(c *gin.Context)         { h.notImplemented(c) }
func (h *Handler) UpdateProfile(c *gin.Context)               { h.notImplemented(c) }
func (h *Handler) GetStatistics(c *gin.Context)               { h.notImplemented(c) }
func (h *Handler) UpdateBooking(c *gin.Context)               { h.notImplemented(c) }
func (h *Handler) CancelBooking(c *gin.Context)               { h.notImplemented(c) }
func (h *Handler) CompleteBooking(c *gin.Context)             { h.notImplemented(c) }
func (h *Handler) GetServices(c *gin.Context) {
    tenantID := c.Query("tenant_id")
    if tenantID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "missing_tenant_id",
            "message": "tenant_id query parameter is required",
        })
        return
    }
    
    locationID := c.Query("location_id")
    category := c.Query("category")
    activeOnly := c.Query("active_only") == "true"
    
    language := c.GetString("language")
    
    // Call booking service via gRPC
    bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])
    
    grpcReq := &bookingPb.GetServicesRequest{
        TenantId:   tenantID,
        LocationId: locationID,
        Category:   category,
        ActiveOnly: activeOnly,
    }
    
    grpcResp, err := bookingClient.GetServices(c.Request.Context(), grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.InvalidArgument:
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": "invalid_argument",
                    "message": st.Message(),
                })
                return
            case codes.NotFound:
                c.JSON(http.StatusNotFound, gin.H{
                    "error": "not_found",
                    "message": st.Message(),
                })
                return
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "services": grpcResp.Services,
    })
}
func (h *Handler) CreateService(c *gin.Context)               { h.notImplemented(c) }
func (h *Handler) UpdateService(c *gin.Context)               { h.notImplemented(c) }
func (h *Handler) DeleteService(c *gin.Context)               { h.notImplemented(c) }
func (h *Handler) GetMasters(c *gin.Context)                  { h.notImplemented(c) }
func (h *Handler) CreateMaster(c *gin.Context)                { h.notImplemented(c) }
func (h *Handler) UpdateMaster(c *gin.Context)                { h.notImplemented(c) }
func (h *Handler) DeleteMaster(c *gin.Context)                { h.notImplemented(c) }
func (h *Handler) GetLocations(c *gin.Context)                { h.notImplemented(c) }
func (h *Handler) CreateLocation(c *gin.Context)              { h.notImplemented(c) }
func (h *Handler) UpdateLocation(c *gin.Context)              { h.notImplemented(c) }
func (h *Handler) DeleteLocation(c *gin.Context)              { h.notImplemented(c) }
func (h *Handler) GetPermissionRequests(c *gin.Context)       { h.notImplemented(c) }
func (h *Handler) CreatePermissionRequest(c *gin.Context)     { h.notImplemented(c) }
func (h *Handler) GetPendingPermissionRequests(c *gin.Context) { h.notImplemented(c) }
func (h *Handler) HandlePermissionRequest(c *gin.Context)     { h.notImplemented(c) }
func (h *Handler) GetBusinessSettings(c *gin.Context)         { h.notImplemented(c) }
func (h *Handler) UpdateBusinessSettings(c *gin.Context)      { h.notImplemented(c) }
func (h *Handler) SuspendTenant(c *gin.Context) {
    var req struct {
        TenantID string `json:"tenant_id" binding:"required"`
        Reason   string `json:"reason" binding:"required"`
        AdminID  string `json:"admin_id" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    language := c.GetString("language")
    
    // Call admin service via gRPC
    adminClient := adminPb.NewAdminServiceClient(h.grpc["admin"])
    
    grpcReq := &adminPb.SuspendTenantRequest{
        TenantId: req.TenantID,
        Reason:   req.Reason,
        AdminId:  req.AdminID,
    }
    
    grpcResp, err := adminClient.SuspendTenant(c.Request.Context(), grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.InvalidArgument:
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": "invalid_argument",
                    "message": st.Message(),
                })
                return
            case codes.NotFound:
                c.JSON(http.StatusNotFound, gin.H{
                    "error": "not_found",
                    "message": st.Message(),
                })
                return
            case codes.FailedPrecondition:
                c.JSON(http.StatusConflict, gin.H{
                    "error": "failed_precondition",
                    "message": st.Message(),
                })
                return
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": grpcResp.Message,
    })
}

func (h *Handler) ReactivateTenant(c *gin.Context) {
    var req struct {
        TenantID string `json:"tenant_id" binding:"required"`
        Notes    string `json:"notes"`
        AdminID  string `json:"admin_id" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "validation_error",
            "message": err.Error(),
        })
        return
    }
    
    language := c.GetString("language")
    
    // Call admin service via gRPC
    adminClient := adminPb.NewAdminServiceClient(h.grpc["admin"])
    
    grpcReq := &adminPb.ReactivateTenantRequest{
        TenantId: req.TenantID,
        Notes:    req.Notes,
        AdminId:  req.AdminID,
    }
    
    grpcResp, err := adminClient.ReactivateTenant(c.Request.Context(), grpcReq)
    if err != nil {
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.InvalidArgument:
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": "invalid_argument",
                    "message": st.Message(),
                })
                return
            case codes.NotFound:
                c.JSON(http.StatusNotFound, gin.H{
                    "error": "not_found",
                    "message": st.Message(),
                })
                return
            case codes.FailedPrecondition:
                c.JSON(http.StatusConflict, gin.H{
                    "error": "failed_precondition",
                    "message": st.Message(),
                })
                return
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal_error",
                    "message": i18n.T(language, "error.internal_server_error"),
                })
                return
            }
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "internal_error",
            "message": i18n.T(language, "error.internal_server_error"),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": grpcResp.Message,
    })
}
func (h *Handler) GetPlatformStatistics(c *gin.Context)       { h.notImplemented(c) }
func (h *Handler) GetSystemHealth(c *gin.Context)             { h.notImplemented(c) }
func (h *Handler) GetSystemLogs(c *gin.Context)               { h.notImplemented(c) }

func (h *Handler) notImplemented(c *gin.Context) {
    language := c.GetString("language")
    c.JSON(http.StatusNotImplemented, gin.H{
        "error": "not_implemented",
        "message": i18n.T(language, "error.not_implemented"),
    })
}
