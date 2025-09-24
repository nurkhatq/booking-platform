package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	adminPb "booking-platform/admin-service/proto"
	bookingPb "booking-platform/booking-service/proto"
	"booking-platform/shared/i18n"
	"booking-platform/shared/models"
	userPb "booking-platform/user-service/proto"
)

type Handler struct {
	grpc map[string]*grpc.ClientConn
}

// Helper function to safely convert *string to string
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")

	// Call user service via gRPC to authenticate
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	grpcResp, err := userClient.Login(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_credentials",
					"message": i18n.T(language, "auth.login.invalid_credentials"),
				})
				return
			case codes.NotFound:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "user_not_found",
					"message": i18n.T(language, "auth.login.user_not_found"),
				})
				return
			case codes.DeadlineExceeded:
				c.JSON(http.StatusRequestTimeout, gin.H{
					"error":   "timeout",
					"message": i18n.T(language, "error.timeout"),
				})
				return
			case codes.Unavailable:
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "service_unavailable",
					"message": i18n.T(language, "error.service_unavailable"),
				})
				return
			default:
				// Log the actual error for debugging
				c.Header("X-Debug-Error", st.Message())
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_error",
					"message": i18n.T(language, "error.internal_server_error"),
				})
				return
			}
		}

		// Non-gRPC error - log for debugging
		c.Header("X-Debug-Error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
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
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")

	// Call user service via gRPC to authenticate
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	grpcResp, err := userClient.Login(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_credentials",
					"message": i18n.T(language, "auth.login.invalid_credentials"),
				})
				return
			case codes.NotFound:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "user_not_found",
					"message": i18n.T(language, "auth.login.user_not_found"),
				})
				return
			case codes.DeadlineExceeded:
				c.JSON(http.StatusRequestTimeout, gin.H{
					"error":   "timeout",
					"message": i18n.T(language, "error.timeout"),
				})
				return
			case codes.Unavailable:
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "service_unavailable",
					"message": i18n.T(language, "error.service_unavailable"),
				})
				return
			default:
				// Log the actual error for debugging
				c.Header("X-Debug-Error", st.Message())
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_error",
					"message": i18n.T(language, "error.internal_server_error"),
				})
				return
			}
		}

		// Non-gRPC error - log for debugging
		c.Header("X-Debug-Error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	// Check if user has SUPER_ADMIN role
	if grpcResp.User.Role != "SUPER_ADMIN" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "insufficient_permissions",
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

// Public API handlers

func (h *Handler) GetBusinessInfo(c *gin.Context) {
	subdomain := c.Param("subdomain")
	language := c.GetString("language")

	// Call user service to get business info
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.GetBusinessInfoRequest{
		Subdomain: subdomain,
	}

	grpcResp, err := userClient.GetBusinessInfo(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "business_not_found",
					"message": i18n.T(language, "business.not_found"),
				})
				return
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_error",
					"message": i18n.T(language, "error.internal_server_error"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"business": gin.H{
			"business_name": grpcResp.BusinessName,
			"business_type": grpcResp.BusinessType,
			"timezone":      grpcResp.Timezone,
			"locations":     grpcResp.Locations,
		},
		"message": i18n.T(language, "business.info_retrieved"),
	})
}

func (h *Handler) GetBusinessLocations(c *gin.Context) {
	subdomain := c.Param("subdomain")
	language := c.GetString("language")

	// Call user service to get business info (which includes locations)
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.GetBusinessInfoRequest{
		Subdomain: subdomain,
	}

	grpcResp, err := userClient.GetBusinessInfo(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"locations": grpcResp.Locations,
		"message":   i18n.T(language, "business.locations_retrieved"),
	})
}

func (h *Handler) GetBusinessServices(c *gin.Context) {
	_ = c.Param("subdomain") // TODO: Use subdomain to get tenant_id
	language := c.GetString("language")

	// Call booking service to get business services
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.GetServicesRequest{
		TenantId:   "", // We need tenant_id, but we have subdomain
		ActiveOnly: true,
	}

	grpcResp, err := bookingClient.GetServices(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"services": grpcResp.Services,
		"message":  i18n.T(language, "business.services_retrieved"),
	})
}

func (h *Handler) GetBusinessMasters(c *gin.Context) {
	_ = c.Param("subdomain") // TODO: Use subdomain to get tenant_id
	language := c.GetString("language")

	// Call user service to get masters
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.GetMastersRequest{
		TenantId: "", // We need tenant_id, but we have subdomain
	}

	grpcResp, err := userClient.GetMasters(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"masters": grpcResp.Masters,
		"message": i18n.T(language, "business.masters_retrieved"),
	})
}

func (h *Handler) CheckAvailability(c *gin.Context) {
	_ = c.Param("subdomain") // TODO: Use subdomain to get tenant_id
	language := c.GetString("language")

	// Get query parameters
	masterID := c.Query("master_id")
	serviceID := c.Query("service_id")
	date := c.Query("date")

	if masterID == "" || serviceID == "" || date == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_parameters",
			"message": i18n.T(language, "booking.missing_parameters"),
		})
		return
	}

	// Call booking service to check availability
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.CheckAvailabilityRequest{
		TenantId:  "", // We need tenant_id, but we have subdomain
		MasterId:  masterID,
		ServiceId: serviceID,
		Date:      date,
	}

	grpcResp, err := bookingClient.CheckAvailability(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"availability": grpcResp.AvailableSlots,
		"message":      i18n.T(language, "booking.availability_checked"),
	})
}

// Client booking handlers

func (h *Handler) VerifyClient(c *gin.Context) {
	var req models.ClientVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")

	// Call user service to create client session
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.CreateClientSessionRequest{
		Email: req.Email,
		Phone: req.Phone,
		Name:  req.Name,
	}

	grpcResp, err := userClient.CreateClientSession(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": grpcResp.SessionId,
		"message":    i18n.T(language, "client.verification_sent"),
	})
}

func (h *Handler) VerifyClientCode(c *gin.Context) {
	var req models.VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")

	// Call user service to verify code
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.VerifyClientCodeRequest{
		SessionId: req.SessionID,
		Code:      req.Code,
	}

	grpcResp, err := userClient.VerifyClientCode(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_code",
					"message": i18n.T(language, "client.invalid_code"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   grpcResp.Token,
		"message": i18n.T(language, "client.verification_success"),
	})
}

func (h *Handler) CreatePublicBooking(c *gin.Context) {
	var req models.BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")

	// Call booking service to create booking
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.CreateBookingRequest{
		TenantId:    "", // We need tenant_id, but we have subdomain
		MasterId:    req.MasterID.String(),
		ServiceId:   req.ServiceID.String(),
		LocationId:  req.LocationID.String(),
		BookingDate: req.BookingDate,
		BookingTime: req.BookingTime,
		ClientName:  req.ClientName,
		ClientEmail: req.ClientEmail,
		ClientPhone: req.ClientPhone,
		ClientNotes: getStringValue(req.ClientNotes),
	}

	grpcResp, err := bookingClient.CreateBooking(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_booking",
					"message": i18n.T(language, "booking.invalid_booking"),
				})
				return
			case codes.Unavailable:
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "service_unavailable",
					"message": i18n.T(language, "error.service_unavailable"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"booking_id":        grpcResp.BookingId,
		"confirmation_code": grpcResp.ConfirmationCode,
		"message":           i18n.T(language, "booking.created_success"),
	})
}

func (h *Handler) GetClientBookings(c *gin.Context) {
	language := c.GetString("language")
	sessionID := c.GetString("client_session_id")

	if sessionID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Get client email from session to filter bookings
	userClient := userPb.NewUserServiceClient(h.grpc["user"])
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	clientSessionReq := &userPb.GetClientSessionRequest{
		SessionId: sessionID,
	}
	clientSessionResp, err := userClient.GetClientSession(ctx, clientSessionReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	// Call booking service to get client bookings
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	grpcReq := &bookingPb.GetBookingsRequest{
		ClientEmail: clientSessionResp.Email,
	}

	grpcResp, err := bookingClient.GetBookings(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bookings": grpcResp.Bookings,
		"message":  i18n.T(language, "booking.list_retrieved"),
	})
}

func (h *Handler) GetClientBooking(c *gin.Context) {
	bookingID := c.Param("id")
	language := c.GetString("language")
	sessionID := c.GetString("client_session_id")

	if sessionID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Call booking service to get specific booking
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.GetBookingRequest{
		BookingId: bookingID,
	}

	grpcResp, err := bookingClient.GetBooking(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "booking_not_found",
					"message": i18n.T(language, "booking.not_found"),
				})
				return
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "access_denied",
					"message": i18n.T(language, "booking.access_denied"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"booking": grpcResp.Booking,
		"message": i18n.T(language, "booking.retrieved"),
	})
}

func (h *Handler) CancelClientBooking(c *gin.Context) {
	bookingID := c.Param("id")
	language := c.GetString("language")
	sessionID := c.GetString("client_session_id")

	if sessionID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	// Call booking service to cancel booking
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.CancelBookingRequest{
		BookingId:   bookingID,
		Reason:      req.Reason,
		CancelledBy: "CLIENT", // Client cancelled
	}

	_, err := bookingClient.CancelBooking(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "booking_not_found",
					"message": i18n.T(language, "booking.not_found"),
				})
				return
			case codes.FailedPrecondition:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "cannot_cancel",
					"message": i18n.T(language, "booking.cannot_cancel"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(language, "booking.cancelled"),
	})
}

func (h *Handler) UpdateClientProfile(c *gin.Context) {
	// TODO: Implement client profile update functionality
	// This would require adding UpdateClientProfile method to the user service
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Client profile update not implemented yet",
	})
}

// Authentication handlers

func (h *Handler) RegisterBusiness(c *gin.Context) {
	var req models.TenantRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")

	// Call user service to register business
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.RegisterBusinessRequest{
		BusinessName: req.BusinessName,
		BusinessType: string(req.BusinessType),
		Subdomain:    req.Subdomain,
		Timezone:     req.Timezone,
		OwnerEmail:   req.OwnerEmail,
		OwnerPhone:   req.OwnerPhone,
		OwnerName:    req.OwnerName,
	}

	grpcResp, err := userClient.RegisterBusiness(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.AlreadyExists:
				c.JSON(http.StatusConflict, gin.H{
					"error":   "business_exists",
					"message": i18n.T(language, "business.already_exists"),
				})
				return
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_data",
					"message": i18n.T(language, "business.invalid_data"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tenant_id": grpcResp.TenantId,
		"message":   i18n.T(language, "business.registration_success"),
	})
}

func (h *Handler) Logout(c *gin.Context) {
	language := c.GetString("language")
	userID := c.GetString("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Call user service to logout
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.LogoutRequest{
		Token: userID, // Assuming userID is the token
	}

	_, err := userClient.Logout(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(language, "auth.logout_success"),
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")

	// Call user service to refresh token
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	}

	grpcResp, err := userClient.RefreshToken(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_refresh_token",
					"message": i18n.T(language, "auth.invalid_refresh_token"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   grpcResp.Token,
		"message": i18n.T(language, "auth.token_refreshed"),
	})
}

func (h *Handler) GetProfile(c *gin.Context) {
	language := c.GetString("language")
	userID := c.GetString("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Call user service to get profile
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &userPb.GetUserProfileRequest{
		UserId: userID,
	}

	grpcResp, err := userClient.GetUserProfile(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "user_not_found",
					"message": i18n.T(language, "user.not_found"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    grpcResp.User,
		"message": i18n.T(language, "user.profile_retrieved"),
	})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	language := c.GetString("language")
	userID := c.GetString("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	var req struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Phone     *string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	// Call user service to update profile
	userClient := userPb.NewUserServiceClient(h.grpc["user"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create user object with updated fields
	user := &userPb.User{
		Id:        userID,
		FirstName: getStringValue(req.FirstName),
		LastName:  getStringValue(req.LastName),
		Phone:     getStringValue(req.Phone),
	}

	grpcReq := &userPb.UpdateUserRequest{
		User: user,
	}

	_, err := userClient.UpdateUser(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(language, "user.profile_updated"),
	})
}

// Dashboard handlers

func (h *Handler) GetDashboard(c *gin.Context) {
	// TODO: Implement dashboard functionality
	// This would require adding GetDashboard method to the admin service
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Dashboard functionality not implemented yet",
	})
}

func (h *Handler) GetStatistics(c *gin.Context) {
	language := c.GetString("language")
	userID := c.GetString("user_id")
	tenantID := c.GetString("tenant_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Get query parameters for date range
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Call admin service to get statistics
	adminClient := adminPb.NewAdminServiceClient(h.grpc["admin"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &adminPb.GetTenantStatisticsRequest{
		TenantId: tenantID,
		DateFrom: startDate,
		DateTo:   endDate,
	}

	grpcResp, err := adminClient.GetTenantStatistics(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": grpcResp.Stats,
		"message":    i18n.T(language, "statistics.retrieved"),
	})
}

// Booking management handlers

func (h *Handler) GetBookings(c *gin.Context) {
	language := c.GetString("language")
	userID := c.GetString("user_id")
	tenantID := c.GetString("tenant_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Get query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	masterID := c.Query("master_id")

	// Call booking service to get bookings
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.GetBookingsRequest{
		TenantId: tenantID,
		Page:     int32(page),
		Limit:    int32(limit),
		Status:   status,
		MasterId: masterID,
	}

	grpcResp, err := bookingClient.GetBookings(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bookings": grpcResp.Bookings,
		"total":    grpcResp.Total,
		"page":     page,
		"limit":    limit,
		"message":  i18n.T(language, "booking.list_retrieved"),
	})
}

func (h *Handler) CreateBooking(c *gin.Context) {
	var req models.BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")
	userID := c.GetString("user_id")
	tenantID := c.GetString("tenant_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Call booking service to create booking
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.CreateBookingRequest{
		TenantId:    tenantID,
		MasterId:    req.MasterID.String(),
		ServiceId:   req.ServiceID.String(),
		LocationId:  req.LocationID.String(),
		BookingDate: req.BookingDate,
		BookingTime: req.BookingTime,
		ClientName:  req.ClientName,
		ClientEmail: req.ClientEmail,
		ClientPhone: req.ClientPhone,
		ClientNotes: getStringValue(req.ClientNotes),
		CreatedBy:   userID,
	}

	grpcResp, err := bookingClient.CreateBooking(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_booking",
					"message": i18n.T(language, "booking.invalid_booking"),
				})
				return
			case codes.Unavailable:
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "service_unavailable",
					"message": i18n.T(language, "error.service_unavailable"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"booking_id":        grpcResp.BookingId,
		"confirmation_code": grpcResp.ConfirmationCode,
		"message":           i18n.T(language, "booking.created_success"),
	})
}

func (h *Handler) UpdateBooking(c *gin.Context) {
	bookingID := c.Param("id")
	language := c.GetString("language")
	userID := c.GetString("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	var req struct {
		BookingDate *string `json:"booking_date"`
		BookingTime *string `json:"booking_time"`
		ClientNotes *string `json:"client_notes"`
		MasterNotes *string `json:"master_notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	// Call booking service to update booking
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.UpdateBookingRequest{
		BookingId:   bookingID,
		BookingDate: getStringValue(req.BookingDate),
		BookingTime: getStringValue(req.BookingTime),
		ClientNotes: getStringValue(req.ClientNotes),
		MasterNotes: getStringValue(req.MasterNotes),
	}

	_, err := bookingClient.UpdateBooking(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "booking_not_found",
					"message": i18n.T(language, "booking.not_found"),
				})
				return
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "access_denied",
					"message": i18n.T(language, "booking.access_denied"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(language, "booking.updated"),
	})
}

func (h *Handler) CancelBooking(c *gin.Context) {
	bookingID := c.Param("id")
	language := c.GetString("language")
	userID := c.GetString("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	// Call booking service to cancel booking
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.CancelBookingRequest{
		BookingId:   bookingID,
		Reason:      req.Reason,
		CancelledBy: "ADMIN", // Admin cancelled
	}

	_, err := bookingClient.CancelBooking(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "booking_not_found",
					"message": i18n.T(language, "booking.not_found"),
				})
				return
			case codes.FailedPrecondition:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "cannot_cancel",
					"message": i18n.T(language, "booking.cannot_cancel"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(language, "booking.cancelled"),
	})
}

func (h *Handler) CompleteBooking(c *gin.Context) {
	bookingID := c.Param("id")
	language := c.GetString("language")
	userID := c.GetString("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	var req struct {
		MasterNotes *string `json:"master_notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	// Call booking service to complete booking
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.CompleteBookingRequest{
		BookingId:   bookingID,
		MasterNotes: getStringValue(req.MasterNotes),
	}

	_, err := bookingClient.CompleteBooking(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "booking_not_found",
					"message": i18n.T(language, "booking.not_found"),
				})
				return
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "access_denied",
					"message": i18n.T(language, "booking.access_denied"),
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(language, "booking.completed"),
	})
}
