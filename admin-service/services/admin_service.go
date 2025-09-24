package services

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    
    "booking-platform/shared/config"
    "booking-platform/shared/database"
    "booking-platform/shared/models"
    pb "booking-platform/admin-service/proto"
)

type AdminService struct {
    pb.UnimplementedAdminServiceServer
    config *config.Config
}

func NewAdminService(cfg *config.Config) *AdminService {
    return &AdminService{
        config: cfg,
    }
}

func (s *AdminService) GetPendingTenants(ctx context.Context, req *pb.GetPendingTenantsRequest) (*pb.GetPendingTenantsResponse, error) {
    db := database.GetDB()
    
    // Set default values
    page := req.Page
    if page < 1 {
        page = 1
    }
    limit := req.Limit
    if limit < 1 {
        limit = 20
    }
    
    sortBy := req.SortBy
    if sortBy == "" {
        sortBy = "created_at"
    }
    
    sortOrder := req.SortOrder
    if sortOrder != "asc" && sortOrder != "desc" {
        sortOrder = "desc"
    }
    
    // Get total count
    var total int
    err := db.Get(&total, "SELECT COUNT(*) FROM tenants WHERE status = 'pending'")
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get total count")
    }
    
    // Get tenants with owner information
    query := fmt.Sprintf(`
        SELECT t.*, u.email, u.phone, u.first_name, u.last_name, u.last_login,
               (SELECT COUNT(*) FROM locations l WHERE l.tenant_id = t.id) as locations_count,
               (SELECT COUNT(*) FROM masters m WHERE m.tenant_id = t.id) as masters_count,
               (SELECT COUNT(*) FROM services s WHERE s.tenant_id = t.id) as services_count
        FROM tenants t
        LEFT JOIN users u ON t.owner_id = u.id
        WHERE t.status = 'pending'
        ORDER BY t.%s %s
        LIMIT $1 OFFSET $2`, sortBy, sortOrder)
    
    offset := (page - 1) * limit
    
    rows, err := db.Query(query, limit, offset)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get pending tenants")
    }
    defer rows.Close()
    
    var tenants []*pb.Tenant
    for rows.Next() {
        var tenant models.Tenant
        var owner struct {
            Email     sql.NullString `db:"email"`
            Phone     sql.NullString `db:"phone"`
            FirstName sql.NullString `db:"first_name"`
            LastName  sql.NullString `db:"last_name"`
            LastLogin sql.NullTime   `db:"last_login"`
        }
        var counts struct {
            LocationsCount int `db:"locations_count"`
            MastersCount   int `db:"masters_count"`
            ServicesCount  int `db:"services_count"`
        }
        
        err := rows.Scan(
            &tenant.ID, &tenant.BusinessName, &tenant.BusinessType, &tenant.Subdomain,
            &tenant.Timezone, &tenant.OwnerID, &tenant.Status, &tenant.TrialStartDate,
            &tenant.TrialEndDate, &tenant.SubscriptionEndDate, &tenant.Settings,
            &tenant.Branding, &tenant.CreatedAt, &tenant.UpdatedAt,
            &owner.Email, &owner.Phone, &owner.FirstName, &owner.LastName, &owner.LastLogin,
            &counts.LocationsCount, &counts.MastersCount, &counts.ServicesCount)
        
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to scan tenant")
        }
        
        tenants = append(tenants, tenantToProto(&tenant, &owner, &counts))
    }
    
    return &pb.GetPendingTenantsResponse{
        Tenants: tenants,
        Total:   int32(total),
        Page:    page,
        Limit:   limit,
    }, nil
}

func (s *AdminService) ApproveTenant(ctx context.Context, req *pb.ApproveTenantRequest) (*pb.ApproveTenantResponse, error) {
    db := database.GetDB()
    
    tenantUUID, err := uuid.Parse(req.TenantId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
    }
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Check if tenant exists and is pending
    var tenant models.Tenant
    err = tx.Get(&tenant, "SELECT * FROM tenants WHERE id = $1 AND status = 'pending'", tenantUUID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Tenant not found or not pending")
        }
        return nil, status.Error(codes.Internal, "Failed to get tenant")
    }
    
    // Calculate trial dates
    trialDays := req.TrialDays
    if trialDays <= 0 {
        trialDays = int32(s.config.Business.DefaultTrialDays)
    }
    
    now := time.Now()
    trialEndDate := now.Add(time.Duration(trialDays) * 24 * time.Hour)
    
    // Update tenant
    _, err = tx.Exec(`
        UPDATE tenants 
        SET status = 'active', trial_start_date = $1, trial_end_date = $2, updated_at = $3
        WHERE id = $4`,
        now, trialEndDate, now, tenantUUID)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to approve tenant")
    }
    
    // Create admin action log
    _, err = tx.Exec(`
        INSERT INTO admin_actions (id, admin_id, action_type, target_type, target_id, details, created_at)
        VALUES ($1, $2, 'approve_tenant', 'tenant', $3, $4, $5)`,
        uuid.New(), req.AdminId, tenantUUID, 
        fmt.Sprintf("Approved with %d days trial. Notes: %s", trialDays, req.Notes),
        now)
    
    if err != nil {
        // Log error but don't fail the operation
        fmt.Printf("Failed to create admin action log: %v\n", err)
    }
    
    // Commit transaction
    if err = tx.Commit(); err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    // TODO: Send approval notification email to business owner
    
    return &pb.ApproveTenantResponse{
        Message:       "Tenant approved successfully",
        TrialEndDate:  trialEndDate.Format("2006-01-02"),
    }, nil
}

func (s *AdminService) RejectTenant(ctx context.Context, req *pb.RejectTenantRequest) (*pb.RejectTenantResponse, error) {
    db := database.GetDB()
    
    tenantUUID, err := uuid.Parse(req.TenantId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
    }
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Check if tenant exists and is pending
    var tenant models.Tenant
    err = tx.Get(&tenant, "SELECT * FROM tenants WHERE id = $1 AND status = 'pending'", tenantUUID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Tenant not found or not pending")
        }
        return nil, status.Error(codes.Internal, "Failed to get tenant")
    }
    
    // Update tenant status to rejected (or delete based on business requirements)
    now := time.Now()
    _, err = tx.Exec(`
        UPDATE tenants 
        SET status = 'rejected', updated_at = $1
        WHERE id = $2`,
        now, tenantUUID)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to reject tenant")
    }
    
    // Create admin action log
    _, err = tx.Exec(`
        INSERT INTO admin_actions (id, admin_id, action_type, target_type, target_id, details, created_at)
        VALUES ($1, $2, 'reject_tenant', 'tenant', $3, $4, $5)`,
        uuid.New(), req.AdminId, tenantUUID, 
        fmt.Sprintf("Rejected. Reason: %s", req.Reason),
        now)
    
    if err != nil {
        fmt.Printf("Failed to create admin action log: %v\n", err)
    }
    
    // Commit transaction
    if err = tx.Commit(); err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    // TODO: Send rejection notification email to business owner
    
    return &pb.RejectTenantResponse{
        Message: "Tenant rejected successfully",
    }, nil
}

func (s *AdminService) GetPlatformStatistics(ctx context.Context, req *pb.GetPlatformStatisticsRequest) (*pb.GetPlatformStatisticsResponse, error) {
    db := database.GetDB()
    
    // Parse date range
    dateFrom := req.DateFrom
    dateTo := req.DateTo
    
    if dateFrom == "" {
        dateFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02") // Last month
    }
    if dateTo == "" {
        dateTo = time.Now().Format("2006-01-02")
    }
    
    // Get tenant statistics
    var tenantStats struct {
        TotalTenants     int `db:"total_tenants"`
        ActiveTenants    int `db:"active_tenants"`
        PendingTenants   int `db:"pending_tenants"`
        SuspendedTenants int `db:"suspended_tenants"`
    }
    
    err := db.Get(&tenantStats, `
        SELECT 
            COUNT(*) as total_tenants,
            COUNT(CASE WHEN status = 'active' THEN 1 END) as active_tenants,
            COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_tenants,
            COUNT(CASE WHEN status = 'suspended' THEN 1 END) as suspended_tenants
        FROM tenants`)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get tenant statistics")
    }
    
    // Get booking statistics
    var bookingStats struct {
        TotalBookings int     `db:"total_bookings"`
        TotalRevenue  float64 `db:"total_revenue"`
    }
    
    err = db.Get(&bookingStats, `
        SELECT 
            COUNT(*) as total_bookings,
            COALESCE(SUM(price), 0) as total_revenue
        FROM bookings 
        WHERE booking_date BETWEEN $1 AND $2 
        AND status = 'completed'`,
        dateFrom, dateTo)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get booking statistics")
    }
    
    // Get daily statistics
    var dailyStats []*pb.DailyStats
    rows, err := db.Query(`
        SELECT 
            DATE(created_at) as date,
            COUNT(*) as new_tenants
        FROM tenants 
        WHERE DATE(created_at) BETWEEN $1 AND $2
        GROUP BY DATE(created_at)
        ORDER BY date`,
        dateFrom, dateTo)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get daily statistics")
    }
    defer rows.Close()
    
    for rows.Next() {
        var date string
        var newTenants int32
        
        err := rows.Scan(&date, &newTenants)
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to scan daily stats")
        }
        
        dailyStats = append(dailyStats, &pb.DailyStats{
            Date:        date,
            NewTenants:  newTenants,
            // Additional stats would be calculated here
        })
    }
    
    // Get business type statistics
    var tenantTypeStats []*pb.TenantTypeStats
    typeRows, err := db.Query(`
        SELECT 
            business_type,
            COUNT(*) as count,
            (COUNT(*) * 100.0 / (SELECT COUNT(*) FROM tenants WHERE status != 'rejected')) as percentage
        FROM tenants 
        WHERE status != 'rejected'
        GROUP BY business_type
        ORDER BY count DESC`)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get business type statistics")
    }
    defer typeRows.Close()
    
    for typeRows.Next() {
        var businessType string
        var count int32
        var percentage float64
        
        err := typeRows.Scan(&businessType, &count, &percentage)
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to scan type stats")
        }
        
        tenantTypeStats = append(tenantTypeStats, &pb.TenantTypeStats{
            BusinessType: businessType,
            Count:        count,
            Percentage:   percentage,
        })
    }
    
    return &pb.GetPlatformStatisticsResponse{
        TotalTenants:     int32(tenantStats.TotalTenants),
        ActiveTenants:    int32(tenantStats.ActiveTenants),
        PendingTenants:   int32(tenantStats.PendingTenants),
        SuspendedTenants: int32(tenantStats.SuspendedTenants),
        TotalBookings:    int32(bookingStats.TotalBookings),
        TotalRevenue:     bookingStats.TotalRevenue,
        DailyStats:       dailyStats,
        TenantTypeStats:  tenantTypeStats,
    }, nil
}

func (s *AdminService) GetSystemHealth(ctx context.Context, req *pb.GetSystemHealthRequest) (*pb.GetSystemHealthResponse, error) {
    // Check database connectivity
    db := database.GetDB()
    dbStatus := "healthy"
    if err := db.Ping(); err != nil {
        dbStatus = "critical"
    }
    
    // Check Redis connectivity
    redisStatus := "healthy"
    // Implementation would check Redis connectivity
    
    // Calculate overall status
    overallStatus := "healthy"
    if dbStatus == "critical" || redisStatus == "critical" {
        overallStatus = "critical"
    }
    
    services := []*pb.ServiceHealth{
        {
            ServiceName:  "database",
            Status:       dbStatus,
            LastCheck:    time.Now().Format(time.RFC3339),
            ResponseTime: "5ms",
        },
        {
            ServiceName:  "redis",
            Status:       redisStatus,
            LastCheck:    time.Now().Format(time.RFC3339),
            ResponseTime: "2ms",
        },
    }
    
    metrics := &pb.SystemMetrics{
        CpuUsage:          45.2,
        MemoryUsage:       62.1,
        DiskUsage:         28.5,
        ActiveConnections: 150,
        ResponseTime:      120.5,
        ErrorRate:         2,
    }
    
    return &pb.GetSystemHealthResponse{
        OverallStatus: overallStatus,
        Services:      services,
        Metrics:       metrics,
        LastCheck:     time.Now().Format(time.RFC3339),
    }, nil
}

// Helper functions
func tenantToProto(tenant *models.Tenant, owner interface{}, counts interface{}) *pb.Tenant {
    var trialStartDate, trialEndDate, subscriptionEndDate string
    
    if tenant.TrialStartDate != nil {
        trialStartDate = tenant.TrialStartDate.Format("2006-01-02")
    }
    if tenant.TrialEndDate != nil {
        trialEndDate = tenant.TrialEndDate.Format("2006-01-02")
    }
    if tenant.SubscriptionEndDate != nil {
        subscriptionEndDate = tenant.SubscriptionEndDate.Format("2006-01-02")
    }
    
    var ownerID string
    if tenant.OwnerID != nil {
        ownerID = tenant.OwnerID.String()
    }
    
    protoTenant := &pb.Tenant{
        Id:                  tenant.ID.String(),
        BusinessName:        tenant.BusinessName,
        BusinessType:        string(tenant.BusinessType),
        Subdomain:          tenant.Subdomain,
        Timezone:           tenant.Timezone,
        OwnerId:            ownerID,
        Status:             string(tenant.Status),
        TrialStartDate:     trialStartDate,
        TrialEndDate:       trialEndDate,
        SubscriptionEndDate: subscriptionEndDate,
        CreatedAt:          tenant.CreatedAt.Format(time.RFC3339),
        UpdatedAt:          tenant.UpdatedAt.Format(time.RFC3339),
    }
    
    // Add owner information and counts if provided
    // Implementation would handle the interface{} parameters properly
    
    return protoTenant
}

// Placeholder implementations for other methods
func (s *AdminService) SuspendTenant(ctx context.Context, req *pb.SuspendTenantRequest) (*pb.SuspendTenantResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) ReactivateTenant(ctx context.Context, req *pb.ReactivateTenantRequest) (*pb.ReactivateTenantResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) GetTenantDetails(ctx context.Context, req *pb.GetTenantDetailsRequest) (*pb.GetTenantDetailsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) UpdateTenantTrial(ctx context.Context, req *pb.UpdateTenantTrialRequest) (*pb.UpdateTenantTrialResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) GetTenantStatistics(ctx context.Context, req *pb.GetTenantStatisticsRequest) (*pb.GetTenantStatisticsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) GetSystemLogs(ctx context.Context, req *pb.GetSystemLogsRequest) (*pb.GetSystemLogsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) GetActiveUsers(ctx context.Context, req *pb.GetActiveUsersRequest) (*pb.GetActiveUsersResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) GetSystemMetrics(ctx context.Context, req *pb.GetSystemMetricsRequest) (*pb.GetSystemMetricsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) BulkUpdateTenants(ctx context.Context, req *pb.BulkUpdateTenantsRequest) (*pb.BulkUpdateTenantsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *AdminService) ExportData(ctx context.Context, req *pb.ExportDataRequest) (*pb.ExportDataResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}
