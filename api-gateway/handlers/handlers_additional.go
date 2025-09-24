package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bookingPb "booking-platform/booking-service/proto"
	"booking-platform/shared/i18n"
)

// Helper function to safely convert *float64 to float64
func getFloat64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// Helper function to safely convert *int to int32
func getInt32Value(i *int) int32 {
	if i == nil {
		return 0
	}
	return int32(*i)
}

// Helper function to safely convert *bool to bool
func getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// Service management handlers

func (h *Handler) GetServices(c *gin.Context) {
	language := c.GetString("language")
	tenantID := c.GetString("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Get query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	category := c.Query("category")

	// Call booking service to get services
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.GetServicesRequest{
		TenantId:   tenantID,
		LocationId: "", // Not filtering by location
		Category:   category,
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
		"page":     page,
		"limit":    limit,
		"message":  i18n.T(language, "service.list_retrieved"),
	})
}

func (h *Handler) CreateService(c *gin.Context) {
	var req struct {
		Category     string                 `json:"category" binding:"required"`
		Name         string                 `json:"name" binding:"required"`
		Description  *string                `json:"description"`
		BasePrice    float64                `json:"base_price" binding:"required"`
		BaseDuration int                    `json:"base_duration" binding:"required"`
		Settings     map[string]interface{} `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	language := c.GetString("language")
	tenantID := c.GetString("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Call booking service to create service
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.CreateServiceRequest{
		TenantId:     tenantID,
		Category:     req.Category,
		Name:         req.Name,
		Description:  getStringValue(req.Description),
		BasePrice:    req.BasePrice,
		BaseDuration: int32(req.BaseDuration),
	}

	grpcResp, err := bookingClient.CreateService(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": i18n.T(language, "error.internal_server_error"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"service_id": grpcResp.ServiceId,
		"message":    i18n.T(language, "service.created"),
	})
}

func (h *Handler) UpdateService(c *gin.Context) {
	serviceID := c.Param("id")
	language := c.GetString("language")
	tenantID := c.GetString("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	var req struct {
		Category     *string                `json:"category"`
		Name         *string                `json:"name"`
		Description  *string                `json:"description"`
		BasePrice    *float64               `json:"base_price"`
		BaseDuration *int                   `json:"base_duration"`
		IsActive     *bool                  `json:"is_active"`
		Settings     map[string]interface{} `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	// Call booking service to update service
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.UpdateServiceRequest{
		ServiceId:    serviceID,
		Category:     getStringValue(req.Category),
		Name:         getStringValue(req.Name),
		Description:  getStringValue(req.Description),
		BasePrice:    getFloat64Value(req.BasePrice),
		BaseDuration: getInt32Value(req.BaseDuration),
		IsActive:     getBoolValue(req.IsActive),
	}

	_, err := bookingClient.UpdateService(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "service_not_found",
					"message": i18n.T(language, "service.not_found"),
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
		"message": i18n.T(language, "service.updated"),
	})
}

func (h *Handler) DeleteService(c *gin.Context) {
	serviceID := c.Param("id")
	language := c.GetString("language")
	tenantID := c.GetString("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": i18n.T(language, "auth.unauthorized"),
		})
		return
	}

	// Call booking service to delete service
	bookingClient := bookingPb.NewBookingServiceClient(h.grpc["booking"])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &bookingPb.DeleteServiceRequest{
		ServiceId: serviceID,
	}

	_, err := bookingClient.DeleteService(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "service_not_found",
					"message": i18n.T(language, "service.not_found"),
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
		"message": i18n.T(language, "service.deleted"),
	})
}

// Master management handlers

func (h *Handler) GetMasters(c *gin.Context) {
	// TODO: Implement master management functionality
	// This would require adding GetMasters method to the user service
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Master management functionality not implemented yet",
	})
}

func (h *Handler) CreateMaster(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Master creation functionality not implemented yet",
	})
}

func (h *Handler) UpdateMaster(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Master update functionality not implemented yet",
	})
}

func (h *Handler) DeleteMaster(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Master deletion functionality not implemented yet",
	})
}

// Location management handlers

func (h *Handler) GetLocations(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Location management functionality not implemented yet",
	})
}

func (h *Handler) CreateLocation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Location creation functionality not implemented yet",
	})
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Location update functionality not implemented yet",
	})
}

func (h *Handler) DeleteLocation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Location deletion functionality not implemented yet",
	})
}

// Placeholder handlers for remaining methods
// These would need to be implemented based on the actual proto definitions

func (h *Handler) GetPermissionRequests(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) CreatePermissionRequest(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetPendingPermissionRequests(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) HandlePermissionRequest(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetBusinessSettings(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) UpdateBusinessSettings(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetPendingTenants(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) ApproveTenant(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) RejectTenant(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) SuspendTenant(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) ReactivateTenant(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetPlatformStatistics(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetSystemHealth(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetSystemLogs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}
