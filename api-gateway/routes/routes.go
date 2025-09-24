package routes

import (
    "github.com/gin-gonic/gin"
    
    "booking-platform/api-gateway/handlers"
    "booking-platform/api-gateway/middleware"
    "booking-platform/shared/config"
)

func SetupRoutes(r *gin.Engine, h *handlers.Handler, cfg *config.Config) {
    // Public API routes (no authentication required)
    public := r.Group("/api/v1/public")
    {
        // Business information
        public.GET("/business/:subdomain", h.GetBusinessInfo)
        public.GET("/business/:subdomain/locations", h.GetBusinessLocations)
        public.GET("/business/:subdomain/services", h.GetBusinessServices)
        public.GET("/business/:subdomain/masters", h.GetBusinessMasters)
        public.GET("/business/:subdomain/availability", h.CheckAvailability)
        
        // Client booking
        public.POST("/client/verify", h.VerifyClient)
        public.POST("/client/verify-code", h.VerifyClientCode)
        public.POST("/booking", h.CreatePublicBooking)
    }
    
    // Client authenticated routes
    client := r.Group("/api/v1/client")
    client.Use(middleware.ClientAuth())
    {
        client.GET("/bookings", h.GetClientBookings)
        client.GET("/booking/:id", h.GetClientBooking)
        client.PUT("/booking/:id/cancel", h.CancelClientBooking)
        client.PUT("/profile", h.UpdateClientProfile)
    }
    
    // Main platform routes (jazyl.tech)
    main := r.Group("/api/v1")
    {
        // Business registration
        main.POST("/register", h.RegisterBusiness)
        main.POST("/login", h.Login)
        main.POST("/logout", h.Logout)
        main.POST("/refresh-token", h.RefreshToken)
    }
    
    // Authenticated routes
    auth := r.Group("/api/v1")
    auth.Use(middleware.AuthRequired())
    {
        // User management
        auth.GET("/profile", h.GetProfile)
        auth.PUT("/profile", h.UpdateProfile)
        
        // Dashboard routes (role-based access)
        auth.GET("/dashboard", h.GetDashboard)
        auth.GET("/statistics", h.GetStatistics)
        
        // Booking management
        auth.GET("/bookings", h.GetBookings)
        auth.POST("/booking", h.CreateBooking)
        auth.PUT("/booking/:id", h.UpdateBooking)
        auth.DELETE("/booking/:id", h.CancelBooking)
        auth.POST("/booking/:id/complete", h.CompleteBooking)
        
        // Service management
        auth.GET("/services", h.GetServices)
        auth.POST("/service", middleware.RequireRole("OWNER", "MANAGER"), h.CreateService)
        auth.PUT("/service/:id", middleware.RequireRole("OWNER", "MANAGER"), h.UpdateService)
        auth.DELETE("/service/:id", middleware.RequireRole("OWNER", "MANAGER"), h.DeleteService)
        
        // Master management
        auth.GET("/masters", h.GetMasters)
        auth.POST("/master", middleware.RequireRole("OWNER", "MANAGER"), h.CreateMaster)
        auth.PUT("/master/:id", middleware.RequireRole("OWNER", "MANAGER"), h.UpdateMaster)
        auth.DELETE("/master/:id", middleware.RequireRole("OWNER", "MANAGER"), h.DeleteMaster)
        
        // Location management
        auth.GET("/locations", h.GetLocations)
        auth.POST("/location", middleware.RequireRole("OWNER"), h.CreateLocation)
        auth.PUT("/location/:id", middleware.RequireRole("OWNER"), h.UpdateLocation)
        auth.DELETE("/location/:id", middleware.RequireRole("OWNER"), h.DeleteLocation)
        
        // Permission requests (Master role)
        auth.GET("/permission-requests", middleware.RequireRole("MASTER"), h.GetPermissionRequests)
        auth.POST("/permission-request", middleware.RequireRole("MASTER"), h.CreatePermissionRequest)
        
        // Permission request management (Owner/Manager roles)
        auth.GET("/manage/permission-requests", middleware.RequireRole("OWNER", "MANAGER"), h.GetPendingPermissionRequests)
        auth.PUT("/manage/permission-request/:id", middleware.RequireRole("OWNER", "MANAGER"), h.HandlePermissionRequest)
        
        // Business settings (Owner role)
        auth.GET("/business/settings", middleware.RequireRole("OWNER"), h.GetBusinessSettings)
        auth.PUT("/business/settings", middleware.RequireRole("OWNER"), h.UpdateBusinessSettings)
    }
    
    // Super Admin routes
    admin := r.Group("/api/v1/admin")
    admin.Use(middleware.AuthRequired())
    admin.Use(middleware.RequireRole("SUPER_ADMIN"))
    {
        // Tenant management
        admin.GET("/tenants", h.GetPendingTenants)
        admin.PUT("/tenant/:id/approve", h.ApproveTenant)
        admin.PUT("/tenant/:id/reject", h.RejectTenant)
        admin.PUT("/tenant/:id/suspend", h.SuspendTenant)
        admin.PUT("/tenant/:id/reactivate", h.ReactivateTenant)
        
        // Platform statistics
        admin.GET("/statistics", h.GetPlatformStatistics)
        admin.GET("/system/health", h.GetSystemHealth)
        admin.GET("/logs", h.GetSystemLogs)
    }
}
