package services

import (
    "context"
    "database/sql"
    "fmt"
    "math/rand"
    "strconv"
    "strings"
    "time"
    
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    
    "booking-platform/shared/config"
    "booking-platform/shared/database"
    "booking-platform/shared/models"
    pb "booking-platform/booking-service/proto"
)

type BookingService struct {
    pb.UnimplementedBookingServiceServer
    config *config.Config
}

func NewBookingService(cfg *config.Config) *BookingService {
    return &BookingService{
        config: cfg,
    }
}

func (s *BookingService) CreateBooking(ctx context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error) {
    db := database.GetDB()
    
    // Parse UUIDs
    tenantID, err := uuid.Parse(req.TenantId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
    }
    
    locationID, err := uuid.Parse(req.LocationId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid location ID")
    }
    
    masterID, err := uuid.Parse(req.MasterId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid master ID")
    }
    
    serviceID, err := uuid.Parse(req.ServiceId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid service ID")
    }
    
    clientSessionID, err := uuid.Parse(req.ClientSessionId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid client session ID")
    }
    
    // Parse booking date
    bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid date format")
    }
    
    // Validate time format
    if !isValidTimeFormat(req.BookingTime) {
        return nil, status.Error(codes.InvalidArgument, "Invalid time format")
    }
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Check if time slot is available
    available, err := s.isTimeSlotAvailable(tx, masterID, bookingDate, req.BookingTime)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to check availability")
    }
    if !available {
        return nil, status.Error(codes.FailedPrecondition, "Time slot not available")
    }
    
    // Get service details for price and duration
    var service models.Service
    err = tx.Get(&service, "SELECT * FROM services WHERE id = $1 AND is_active = true", serviceID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Service not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Create booking
    bookingID := uuid.New()
    confirmationCode := generateConfirmationCode()
    
    _, err = tx.Exec(`
        INSERT INTO bookings (
            id, tenant_id, location_id, master_id, service_id, client_session_id,
            client_name, client_email, client_phone, booking_date, booking_time,
            duration, price, status, client_notes, created_by, confirmation_code
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
        bookingID, tenantID, locationID, masterID, serviceID, clientSessionID,
        req.ClientName, req.ClientEmail, req.ClientPhone, bookingDate, req.BookingTime,
        service.BaseDuration, service.BasePrice, models.BookingPending, req.ClientNotes, req.CreatedBy, confirmationCode)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to create booking")
    }
    
    // Clear availability cache
    s.clearAvailabilityCache(masterID, bookingDate)
    
    err = tx.Commit()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    return &pb.CreateBookingResponse{
        BookingId:        bookingID.String(),
        ConfirmationCode: confirmationCode,
        Message:          "Booking created successfully",
    }, nil
}

func (s *BookingService) GetBookings(ctx context.Context, req *pb.GetBookingsRequest) (*pb.GetBookingsResponse, error) {
    db := database.GetDB()
    
    tenantID, err := uuid.Parse(req.TenantId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
    }
    
    // Build query
    query := "SELECT * FROM bookings WHERE tenant_id = $1"
    args := []interface{}{tenantID}
    argIndex := 2
    
    // Add filters
    if req.LocationId != "" {
        locationID, err := uuid.Parse(req.LocationId)
        if err != nil {
            return nil, status.Error(codes.InvalidArgument, "Invalid location ID")
        }
        query += fmt.Sprintf(" AND location_id = $%d", argIndex)
        args = append(args, locationID)
        argIndex++
    }
    
    if req.MasterId != "" {
        masterID, err := uuid.Parse(req.MasterId)
        if err != nil {
            return nil, status.Error(codes.InvalidArgument, "Invalid master ID")
        }
        query += fmt.Sprintf(" AND master_id = $%d", argIndex)
        args = append(args, masterID)
        argIndex++
    }
    
    if req.ClientEmail != "" {
        query += fmt.Sprintf(" AND client_email = $%d", argIndex)
        args = append(args, req.ClientEmail)
        argIndex++
    }
    
    if req.Status != "" {
        query += fmt.Sprintf(" AND status = $%d", argIndex)
        args = append(args, req.Status)
        argIndex++
    }
    
    if req.DateFrom != "" {
        query += fmt.Sprintf(" AND booking_date >= $%d", argIndex)
        args = append(args, req.DateFrom)
        argIndex++
    }
    
    if req.DateTo != "" {
        query += fmt.Sprintf(" AND booking_date <= $%d", argIndex)
        args = append(args, req.DateTo)
        argIndex++
    }
    
    // Add pagination
    limit := req.Limit
    if limit <= 0 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }
    
    offset := (req.Page - 1) * limit
    if offset < 0 {
        offset = 0
    }
    
    query += fmt.Sprintf(" ORDER BY booking_date DESC, booking_time DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
    args = append(args, limit, offset)
    
    // Execute query
    var bookings []models.Booking
    err = db.Select(&bookings, query, args...)
    if err != nil {
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Get total count
    countQuery := strings.Split(query, " ORDER BY")[0]
    countQuery = strings.Replace(countQuery, "SELECT *", "SELECT COUNT(*)", 1)
    countQuery = strings.Replace(countQuery, fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1), "", 1)
    
    var total int32
    err = db.Get(&total, countQuery, args[:len(args)-2]...)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get total count")
    }
    
    // Convert to proto
    var protoBookings []*pb.Booking
    for _, booking := range bookings {
        protoBookings = append(protoBookings, bookingToProto(&booking, nil, nil, nil))
    }
    
    return &pb.GetBookingsResponse{
        Bookings: protoBookings,
        Total:    total,
        Page:     req.Page,
        Limit:    limit,
    }, nil
}

func (s *BookingService) CheckAvailability(ctx context.Context, req *pb.CheckAvailabilityRequest) (*pb.CheckAvailabilityResponse, error) {
    db := database.GetDB()
    
    masterID, err := uuid.Parse(req.MasterId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid master ID")
    }
    
    date, err := time.Parse("2006-01-02", req.Date)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid date format")
    }
    
    // Get available time slots
    slots, err := s.getAvailableTimeSlots(masterID, date)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get available slots")
    }
    
    // Convert to proto
    var protoSlots []*pb.TimeSlot
    for _, slot := range slots {
        protoSlots = append(protoSlots, &pb.TimeSlot{
            StartTime:   slot.StartTime,
            EndTime:     slot.EndTime,
            IsAvailable: slot.IsAvailable,
        })
    }
    
    return &pb.CheckAvailabilityResponse{
        AvailableSlots: protoSlots,
    }, nil
}

func (s *BookingService) CancelBooking(ctx context.Context, req *pb.CancelBookingRequest) (*pb.CancelBookingResponse, error) {
    db := database.GetDB()
    
    bookingID, err := uuid.Parse(req.BookingId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid booking ID")
    }
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Check if booking exists and can be cancelled
    var booking models.Booking
    err = tx.Get(&booking, "SELECT * FROM bookings WHERE id = $1", bookingID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Booking not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    if booking.Status == models.BookingCancelled {
        return nil, status.Error(codes.FailedPrecondition, "Booking already cancelled")
    }
    if booking.Status == models.BookingCompleted {
        return nil, status.Error(codes.FailedPrecondition, "Cannot cancel completed booking")
    }
    
    // Update booking status to cancelled
    _, err = tx.Exec(`
        UPDATE bookings 
        SET status = 'cancelled', 
            cancellation_reason = $1, 
            cancelled_by = $2, 
            cancelled_at = CURRENT_TIMESTAMP,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $3`,
        req.Reason, req.CancelledBy, bookingID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to cancel booking")
    }
    
    // Clear availability cache
    s.clearAvailabilityCache(booking.MasterID, booking.BookingDate)
    
    err = tx.Commit()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    return &pb.CancelBookingResponse{
        Message: "Booking cancelled successfully",
    }, nil
}

func (s *BookingService) isTimeSlotAvailable(tx *sqlx.Tx, masterID uuid.UUID, date time.Time, timeSlot string) (bool, error) {
    // Check if there's already a booking for this master at this time
    var count int
    err := tx.Get(&count, `
        SELECT COUNT(*) FROM bookings 
        WHERE master_id = $1 AND booking_date = $2 AND booking_time = $3 
        AND status IN ('pending', 'confirmed')`,
        masterID, date, timeSlot)
    if err != nil {
        return false, err
    }
    
    return count == 0, nil
}

func (s *BookingService) getAvailableTimeSlots(masterID uuid.UUID, date time.Time) ([]TimeSlot, error) {
    // Generate time slots from 9:00 to 18:00 (30-minute intervals)
    var slots []TimeSlot
    
    for hour := 9; hour < 18; hour++ {
        for minute := 0; minute < 60; minute += 30 {
            startTime := fmt.Sprintf("%02d:%02d", hour, minute)
            endHour := hour
            endMinute := minute + 30
            if endMinute >= 60 {
                endHour++
                endMinute = 0
            }
            endTime := fmt.Sprintf("%02d:%02d", endHour, endMinute)
            
            // Check if slot is available
            db := database.GetDB()
            var count int
            err := db.Get(&count, `
                SELECT COUNT(*) FROM bookings 
                WHERE master_id = $1 AND booking_date = $2 AND booking_time = $3 
                AND status IN ('pending', 'confirmed')`,
                masterID, date, startTime)
            if err != nil {
                return nil, err
            }
            
            slots = append(slots, TimeSlot{
                StartTime:   startTime,
                EndTime:     endTime,
                IsAvailable: count == 0,
            })
        }
    }
    
    return slots, nil
}

func (s *BookingService) clearAvailabilityCache(masterID uuid.UUID, date time.Time) {
    // In a real implementation, you would clear Redis cache here
    // For now, we'll just log it
    fmt.Printf("Clearing availability cache for master %s on %s\n", masterID, date.Format("2006-01-02"))
}

func generateConfirmationCode() string {
    return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func isValidTimeFormat(timeStr string) bool {
    parts := strings.Split(timeStr, ":")
    if len(parts) != 2 {
        return false
    }
    
    hour, err := strconv.Atoi(parts[0])
    if err != nil || hour < 0 || hour > 23 {
        return false
    }
    
    minute, err := strconv.Atoi(parts[1])
    if err != nil || minute < 0 || minute > 59 {
        return false
    }
    
    return true
}

func parseTimeHour(timeStr string) int {
    parts := strings.Split(timeStr, ":")
    hour, _ := strconv.Atoi(parts[0])
    return hour
}

func parseTimeMinute(timeStr string) int {
    parts := strings.Split(timeStr, ":")
    minute, _ := strconv.Atoi(parts[1])
    return minute
}

func bookingToProto(booking *models.Booking, master interface{}, service interface{}, location interface{}) *pb.Booking {
    var clientNotes string
    if booking.ClientNotes != nil {
        clientNotes = *booking.ClientNotes
    }
    
    var masterNotes string
    if booking.MasterNotes != nil {
        masterNotes = *booking.MasterNotes
    }
    
    var cancellationReason string
    if booking.CancellationReason != nil {
        cancellationReason = *booking.CancellationReason
    }
    
    var cancelledBy string
    if booking.CancelledBy != nil {
        cancelledBy = *booking.CancelledBy
    }
    
    var cancelledAt string
    if booking.CancelledAt != nil {
        cancelledAt = booking.CancelledAt.Format(time.RFC3339)
    }
    
    var confirmationCode string
    if booking.ConfirmationCode != nil {
        confirmationCode = *booking.ConfirmationCode
    }
    
    return &pb.Booking{
        Id:                 booking.ID.String(),
        TenantId:          booking.TenantID.String(),
        LocationId:        booking.LocationID.String(),
        MasterId:          booking.MasterID.String(),
        ServiceId:         booking.ServiceID.String(),
        ClientSessionId:   booking.ClientSessionID.String(),
        ClientName:        booking.ClientName,
        ClientEmail:       booking.ClientEmail,
        ClientPhone:       booking.ClientPhone,
        BookingDate:       booking.BookingDate.Format("2006-01-02"),
        BookingTime:       booking.BookingTime,
        Duration:          int32(booking.Duration),
        Price:             booking.Price,
        Status:            string(booking.Status),
        ClientNotes:       clientNotes,
        MasterNotes:       masterNotes,
        CancellationReason: cancellationReason,
        CancelledBy:       cancelledBy,
        CancelledAt:       cancelledAt,
        CreatedBy:         booking.CreatedBy,
        ConfirmationCode:  confirmationCode,
        CreatedAt:         booking.CreatedAt.Format(time.RFC3339),
        UpdatedAt:         booking.UpdatedAt.Format(time.RFC3339),
    }
}

type TimeSlot struct {
    StartTime   string
    EndTime     string
    IsAvailable bool
}

func (s *BookingService) GetBooking(ctx context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error) {
    db := database.GetDB()
    
    bookingID, err := uuid.Parse(req.BookingId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid booking ID")
    }
    
    var booking models.Booking
    err = db.Get(&booking, 
        "SELECT * FROM bookings WHERE id = $1", 
        bookingID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Booking not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Get related data
    var master models.Master
    var service models.Service
    var location models.Location
    
    // Get master info
    err = db.Get(&master, 
        "SELECT m.*, u.email, u.first_name, u.last_name, u.phone FROM masters m "+
        "JOIN users u ON m.user_id = u.id WHERE m.id = $1", 
        booking.MasterID)
    if err != nil && err != sql.ErrNoRows {
        return nil, status.Error(codes.Internal, "Failed to get master info")
    }
    
    // Get service info
    err = db.Get(&service, 
        "SELECT * FROM services WHERE id = $1", 
        booking.ServiceID)
    if err != nil && err != sql.ErrNoRows {
        return nil, status.Error(codes.Internal, "Failed to get service info")
    }
    
    // Get location info
    err = db.Get(&location, 
        "SELECT * FROM locations WHERE id = $1", 
        booking.LocationID)
    if err != nil && err != sql.ErrNoRows {
        return nil, status.Error(codes.Internal, "Failed to get location info")
    }
    
    return &pb.GetBookingResponse{
        Booking: bookingToProto(&booking, &master, &service, &location),
    }, nil
}

func (s *BookingService) UpdateBooking(ctx context.Context, req *pb.UpdateBookingRequest) (*pb.UpdateBookingResponse, error) {
    db := database.GetDB()
    
    bookingID, err := uuid.Parse(req.BookingId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid booking ID")
    }
    
    // Check if booking exists
    var existingBooking models.Booking
    err = db.Get(&existingBooking, 
        "SELECT * FROM bookings WHERE id = $1", 
        bookingID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Booking not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Check if booking can be updated (not completed or cancelled)
    if existingBooking.Status == models.BookingCompleted || existingBooking.Status == models.BookingCancelled {
        return nil, status.Error(codes.FailedPrecondition, "Cannot update completed or cancelled booking")
    }
    
    // Parse new date if provided
    var newDate time.Time
    if req.BookingDate != "" {
        newDate, err = time.Parse("2006-01-02", req.BookingDate)
        if err != nil {
            return nil, status.Error(codes.InvalidArgument, "Invalid date format")
        }
    } else {
        newDate = existingBooking.BookingDate
    }
    
    // Validate time format if provided
    if req.BookingTime != "" && !isValidTimeFormat(req.BookingTime) {
        return nil, status.Error(codes.InvalidArgument, "Invalid time format")
    }
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Check availability if date/time changed
    if req.BookingDate != "" || req.BookingTime != "" {
        newTime := req.BookingTime
        if newTime == "" {
            newTime = existingBooking.BookingTime
        }
        
        available, err := s.isTimeSlotAvailable(tx, existingBooking.MasterID, newDate, newTime)
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to check availability")
        }
        if !available {
            return nil, status.Error(codes.FailedPrecondition, "Time slot not available")
        }
    }
    
    // Update booking
    _, err = tx.Exec(`
        UPDATE bookings 
        SET booking_date = $1, 
            booking_time = $2, 
            client_notes = $3, 
            master_notes = $4,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $5`,
        newDate,
        req.BookingTime,
        req.ClientNotes,
        req.MasterNotes,
        bookingID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to update booking")
    }
    
    // Clear availability cache
    s.clearAvailabilityCache(existingBooking.MasterID, newDate)
    if req.BookingDate != "" {
        s.clearAvailabilityCache(existingBooking.MasterID, existingBooking.BookingDate)
    }
    
    err = tx.Commit()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    return &pb.UpdateBookingResponse{
        Message: "Booking updated successfully",
    }, nil
}

func (s *BookingService) CompleteBooking(ctx context.Context, req *pb.CompleteBookingRequest) (*pb.CompleteBookingResponse, error) {
    db := database.GetDB()
    
    bookingID, err := uuid.Parse(req.BookingId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid booking ID")
    }
    
    // Check if booking exists and can be completed
    var booking models.Booking
    err = db.Get(&booking, 
        "SELECT * FROM bookings WHERE id = $1", 
        bookingID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Booking not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Check if booking can be completed
    if booking.Status == models.BookingCompleted {
        return nil, status.Error(codes.FailedPrecondition, "Booking already completed")
    }
    if booking.Status == models.BookingCancelled {
        return nil, status.Error(codes.FailedPrecondition, "Cannot complete cancelled booking")
    }
    
    // Update booking status to completed
    _, err = db.Exec(`
        UPDATE bookings 
        SET status = 'completed', 
            master_notes = $1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $2`,
        req.MasterNotes,
        bookingID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to complete booking")
    }
    
    return &pb.CompleteBookingResponse{
        Message: "Booking completed successfully",
    }, nil
}

func (s *BookingService) GetMasterSchedule(ctx context.Context, req *pb.GetMasterScheduleRequest) (*pb.GetMasterScheduleResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *BookingService) CreateService(ctx context.Context, req *pb.CreateServiceRequest) (*pb.CreateServiceResponse, error) {
    db := database.GetDB()
    
    tenantID, err := uuid.Parse(req.TenantId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
    }
    
    // Validate required fields
    if req.Name == "" {
        return nil, status.Error(codes.InvalidArgument, "Service name is required")
    }
    if req.BasePrice < 0 {
        return nil, status.Error(codes.InvalidArgument, "Base price cannot be negative")
    }
    if req.BaseDuration <= 0 {
        return nil, status.Error(codes.InvalidArgument, "Base duration must be positive")
    }
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Check if tenant exists
    var tenantExists bool
    err = tx.Get(&tenantExists, "SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", tenantID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Database error")
    }
    if !tenantExists {
        return nil, status.Error(codes.NotFound, "Tenant not found")
    }
    
    // Create service
    serviceID := uuid.New()
    _, err = tx.Exec(`
        INSERT INTO services (id, tenant_id, category, name, description, base_price, base_duration, is_active, popularity_score)
        VALUES ($1, $2, $3, $4, $5, $6, $7, true, 0)`,
        serviceID, tenantID, req.Category, req.Name, req.Description, req.BasePrice, req.BaseDuration)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to create service")
    }
    
    err = tx.Commit()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    return &pb.CreateServiceResponse{
        ServiceId: serviceID.String(),
        Message:   "Service created successfully",
    }, nil
}

func (s *BookingService) UpdateService(ctx context.Context, req *pb.UpdateServiceRequest) (*pb.UpdateServiceResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *BookingService) GetServices(ctx context.Context, req *pb.GetServicesRequest) (*pb.GetServicesResponse, error) {
    db := database.GetDB()
    
    tenantID, err := uuid.Parse(req.TenantId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
    }
    
    // Build query
    query := "SELECT * FROM services WHERE tenant_id = $1"
    args := []interface{}{tenantID}
    argIndex := 2
    
    // Add location filter if provided
    if req.LocationId != "" {
        locationID, err := uuid.Parse(req.LocationId)
        if err != nil {
            return nil, status.Error(codes.InvalidArgument, "Invalid location ID")
        }
        
        query += fmt.Sprintf(" AND id IN (SELECT service_id FROM location_services WHERE location_id = $%d)", argIndex)
        args = append(args, locationID)
        argIndex++
    }
    
    // Add category filter if provided
    if req.Category != "" {
        query += fmt.Sprintf(" AND category = $%d", argIndex)
        args = append(args, req.Category)
        argIndex++
    }
    
    // Add active filter if requested
    if req.ActiveOnly {
        query += fmt.Sprintf(" AND is_active = $%d", argIndex)
        args = append(args, true)
        argIndex++
    }
    
    // Order by popularity and name
    query += " ORDER BY popularity_score DESC, name ASC"
    
    // Execute query
    var services []models.Service
    err = db.Select(&services, query, args...)
    if err != nil {
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Convert to proto
    var protoServices []*pb.Service
    for _, service := range services {
        var description string
        if service.Description != nil {
            description = *service.Description
        }
        
        protoServices = append(protoServices, &pb.Service{
            Id:              service.ID.String(),
            TenantId:        service.TenantID.String(),
            Category:        service.Category,
            Name:            service.Name,
            Description:     description,
            BasePrice:       service.BasePrice,
            BaseDuration:    int32(service.BaseDuration),
            IsActive:        service.IsActive,
            PopularityScore: int32(service.PopularityScore),
        })
    }
    
    return &pb.GetServicesResponse{
        Services: protoServices,
    }, nil
}

func (s *BookingService) DeleteService(ctx context.Context, req *pb.DeleteServiceRequest) (*pb.DeleteServiceResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *BookingService) GetBookingStatistics(ctx context.Context, req *pb.GetBookingStatisticsRequest) (*pb.GetBookingStatisticsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}