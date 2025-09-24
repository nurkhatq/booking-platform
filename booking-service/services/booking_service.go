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
    "booking-platform/shared/cache"
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
    tenantUUID, _ := uuid.Parse(req.TenantId)
    locationUUID, _ := uuid.Parse(req.LocationId)
    masterUUID, _ := uuid.Parse(req.MasterId)
    serviceUUID, _ := uuid.Parse(req.ServiceId)
    clientSessionUUID, _ := uuid.Parse(req.ClientSessionId)
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Parse booking date and time
    bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid booking date format")
    }
    
    // Validate booking time format (HH:MM)
    if !isValidTimeFormat(req.BookingTime) {
        return nil, status.Error(codes.InvalidArgument, "Invalid booking time format")
    }
    
    // Check if the time slot is available
    available, err := s.isTimeSlotAvailable(tx, masterUUID, bookingDate, req.BookingTime)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to check availability")
    }
    if !available {
        return nil, status.Error(codes.AlreadyExists, "Time slot is already booked")
    }
    
    // Get service information for pricing and duration
    var service models.Service
    err = tx.Get(&service, "SELECT * FROM services WHERE id = $1 AND tenant_id = $2", serviceUUID, tenantUUID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Service not found")
        }
        return nil, status.Error(codes.Internal, "Failed to get service")
    }
    
    // Check for master-specific pricing
    var masterService struct {
        Price    *float64 `db:"price"`
        Duration *int     `db:"duration"`
    }
    err = tx.Get(&masterService, 
        "SELECT price, duration FROM master_services WHERE master_id = $1 AND service_id = $2",
        masterUUID, serviceUUID)
    
    price := service.BasePrice
    duration := service.BaseDuration
    
    if err == nil {
        // Use master-specific pricing if available
        if masterService.Price != nil {
            price = *masterService.Price
        }
        if masterService.Duration != nil {
            duration = *masterService.Duration
        }
    }
    
    // Generate confirmation code
    confirmationCode := generateConfirmationCode()
    
    // Create booking
    bookingID := uuid.New()
    _, err = tx.Exec(`
        INSERT INTO bookings (
            id, tenant_id, location_id, master_id, service_id, client_session_id,
            client_name, client_email, client_phone, booking_date, booking_time,
            duration, price, status, client_notes, created_by, confirmation_code
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
        bookingID, tenantUUID, locationUUID, masterUUID, serviceUUID, clientSessionUUID,
        req.ClientName, req.ClientEmail, req.ClientPhone, bookingDate, req.BookingTime,
        duration, price, "confirmed", req.ClientNotes, req.CreatedBy, confirmationCode)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to create booking")
    }
    
    // Update service popularity
    _, err = tx.Exec(
        "UPDATE services SET popularity_score = popularity_score + 1 WHERE id = $1",
        serviceUUID)
    if err != nil {
        // Log error but don't fail the booking
        fmt.Printf("Failed to update service popularity: %v\n", err)
    }
    
    // Commit transaction
    if err = tx.Commit(); err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    // Clear availability cache
    s.clearAvailabilityCache(masterUUID, bookingDate)
    
    // TODO: Send confirmation notification
    
    return &pb.CreateBookingResponse{
        BookingId:        bookingID.String(),
        ConfirmationCode: confirmationCode,
        Message:          "Booking created successfully",
    }, nil
}

func (s *BookingService) GetBookings(ctx context.Context, req *pb.GetBookingsRequest) (*pb.GetBookingsResponse, error) {
    db := database.GetDB()
    
    // Build query dynamically based on filters
    query := `
        SELECT b.*, 
               m.bio, m.photo_url, m.specialization, m.experience_years, m.rating, m.total_reviews,
               u.first_name, u.last_name, u.email as master_email, u.phone as master_phone,
               s.name as service_name, s.category, s.description,
               l.name as location_name, l.address, l.city
        FROM bookings b
        LEFT JOIN masters m ON b.master_id = m.id
        LEFT JOIN users u ON m.user_id = u.id
        LEFT JOIN services s ON b.service_id = s.id
        LEFT JOIN locations l ON b.location_id = l.id
        WHERE 1=1`
    
    args := []interface{}{}
    argCount := 0
    
    if req.TenantId != "" {
        argCount++
        query += fmt.Sprintf(" AND b.tenant_id = $%d", argCount)
        args = append(args, req.TenantId)
    }
    
    if req.LocationId != "" {
        argCount++
        query += fmt.Sprintf(" AND b.location_id = $%d", argCount)
        args = append(args, req.LocationId)
    }
    
    if req.MasterId != "" {
        argCount++
        query += fmt.Sprintf(" AND b.master_id = $%d", argCount)
        args = append(args, req.MasterId)
    }
    
    if req.ClientEmail != "" {
        argCount++
        query += fmt.Sprintf(" AND b.client_email = $%d", argCount)
        args = append(args, req.ClientEmail)
    }
    
    if req.Status != "" {
        argCount++
        query += fmt.Sprintf(" AND b.status = $%d", argCount)
        args = append(args, req.Status)
    }
    
    if req.DateFrom != "" {
        argCount++
        query += fmt.Sprintf(" AND b.booking_date >= $%d", argCount)
        args = append(args, req.DateFrom)
    }
    
    if req.DateTo != "" {
        argCount++
        query += fmt.Sprintf(" AND b.booking_date <= $%d", argCount)
        args = append(args, req.DateTo)
    }
    
    // Get total count
    countQuery := strings.Replace(query, 
        "SELECT b.*, m.bio, m.photo_url, m.specialization, m.experience_years, m.rating, m.total_reviews, u.first_name, u.last_name, u.email as master_email, u.phone as master_phone, s.name as service_name, s.category, s.description, l.name as location_name, l.address, l.city FROM bookings b LEFT JOIN masters m ON b.master_id = m.id LEFT JOIN users u ON m.user_id = u.id LEFT JOIN services s ON b.service_id = s.id LEFT JOIN locations l ON b.location_id = l.id",
        "SELECT COUNT(*) FROM bookings b LEFT JOIN masters m ON b.master_id = m.id LEFT JOIN users u ON m.user_id = u.id LEFT JOIN services s ON b.service_id = s.id LEFT JOIN locations l ON b.location_id = l.id", 1)
    
    var total int
    err := db.Get(&total, countQuery, args...)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get total count")
    }
    
    // Add pagination
    query += " ORDER BY b.booking_date DESC, b.booking_time DESC"
    
    if req.Limit > 0 {
        argCount++
        query += fmt.Sprintf(" LIMIT $%d", argCount)
        args = append(args, req.Limit)
        
        if req.Page > 0 {
            offset := (req.Page - 1) * req.Limit
            argCount++
            query += fmt.Sprintf(" OFFSET $%d", argCount)
            args = append(args, offset)
        }
    }
    
    // Execute query
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get bookings")
    }
    defer rows.Close()
    
    var bookings []*pb.Booking
    for rows.Next() {
        var booking models.Booking
        var master struct {
            Bio             *string  `db:"bio"`
            PhotoURL        *string  `db:"photo_url"`
            Specialization  *string  `db:"specialization"`
            ExperienceYears int      `db:"experience_years"`
            Rating          float64  `db:"rating"`
            TotalReviews    int      `db:"total_reviews"`
            FirstName       *string  `db:"first_name"`
            LastName        *string  `db:"last_name"`
            Email           string   `db:"master_email"`
            Phone           *string  `db:"master_phone"`
        }
        var service struct {
            Name        string  `db:"service_name"`
            Category    string  `db:"category"`
            Description *string `db:"description"`
        }
        var location struct {
            Name    string `db:"location_name"`
            Address string `db:"address"`
            City    string `db:"city"`
        }
        
        err := rows.Scan(
            &booking.ID, &booking.TenantID, &booking.LocationID, &booking.MasterID, &booking.ServiceID,
            &booking.ClientSessionID, &booking.ClientName, &booking.ClientEmail, &booking.ClientPhone,
            &booking.BookingDate, &booking.BookingTime, &booking.Duration, &booking.Price, &booking.Status,
            &booking.ClientNotes, &booking.MasterNotes, &booking.CancellationReason, &booking.CancelledBy,
            &booking.CancelledAt, &booking.CreatedBy, &booking.ConfirmationCode, &booking.ReminderSent24h,
            &booking.ReminderSent2h, &booking.CreatedAt, &booking.UpdatedAt,
            &master.Bio, &master.PhotoURL, &master.Specialization, &master.ExperienceYears,
            &master.Rating, &master.TotalReviews, &master.FirstName, &master.LastName,
            &master.Email, &master.Phone, &service.Name, &service.Category, &service.Description,
            &location.Name, &location.Address, &location.City)
        
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to scan booking")
        }
        
        bookings = append(bookings, bookingToProto(&booking, &master, &service, &location))
    }
    
    return &pb.GetBookingsResponse{
        Bookings: bookings,
        Total:    int32(total),
        Page:     req.Page,
        Limit:    req.Limit,
    }, nil
}

func (s *BookingService) CheckAvailability(ctx context.Context, req *pb.CheckAvailabilityRequest) (*pb.CheckAvailabilityResponse, error) {
    // Parse date
    date, err := time.Parse("2006-01-02", req.Date)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid date format")
    }
    
    masterUUID, _ := uuid.Parse(req.MasterId)
    
    // Check cache first
    cacheKey := fmt.Sprintf("availability:%s:%s", req.MasterId, req.Date)
    var cachedSlots []*pb.TimeSlot
    
    if cache.Exists(ctx, cacheKey) {
        if err := cache.Get(ctx, cacheKey, &cachedSlots); err == nil {
            return &pb.CheckAvailabilityResponse{
                AvailableSlots: cachedSlots,
            }, nil
        }
    }
    
    // Get available time slots
    slots, err := s.getAvailableTimeSlots(masterUUID, date)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get availability")
    }
    
    // Convert to proto
    protoSlots := make([]*pb.TimeSlot, len(slots))
    for i, slot := range slots {
        protoSlots[i] = &pb.TimeSlot{
            StartTime:   slot.StartTime,
            EndTime:     slot.EndTime,
            IsAvailable: slot.IsAvailable,
        }
    }
    
    // Cache for 5 minutes
    cache.Set(ctx, cacheKey, protoSlots, 5*time.Minute)
    
    return &pb.CheckAvailabilityResponse{
        AvailableSlots: protoSlots,
    }, nil
}

func (s *BookingService) CancelBooking(ctx context.Context, req *pb.CancelBookingRequest) (*pb.CancelBookingResponse, error) {
    db := database.GetDB()
    
    bookingUUID, err := uuid.Parse(req.BookingId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid booking ID")
    }
    
    // Get booking details
    var booking models.Booking
    err = db.Get(&booking, "SELECT * FROM bookings WHERE id = $1", bookingUUID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Booking not found")
        }
        return nil, status.Error(codes.Internal, "Failed to get booking")
    }
    
    // Check if booking can be cancelled
    if booking.Status == models.BookingCancelled {
        return nil, status.Error(codes.FailedPrecondition, "Booking is already cancelled")
    }
    
    if booking.Status == models.BookingCompleted {
        return nil, status.Error(codes.FailedPrecondition, "Cannot cancel completed booking")
    }
    
    // Check cancellation policy (2 hours before appointment)
    bookingDateTime := time.Date(
        booking.BookingDate.Year(), booking.BookingDate.Month(), booking.BookingDate.Day(),
        parseTimeHour(booking.BookingTime), parseTimeMinute(booking.BookingTime), 0, 0,
        time.UTC)
    
    if time.Now().Add(time.Duration(s.config.Business.CancellationHours)*time.Hour).After(bookingDateTime) {
        // Only owners/managers can cancel within cancellation window
        if req.CancelledBy != "OWNER" && req.CancelledBy != "MANAGER" {
            return nil, status.Error(codes.FailedPrecondition, 
                fmt.Sprintf("Cannot cancel booking less than %d hours before appointment", s.config.Business.CancellationHours))
        }
    }
    
    // Update booking
    now := time.Now()
    _, err = db.Exec(`
        UPDATE bookings 
        SET status = 'cancelled', cancellation_reason = $1, cancelled_by = $2, cancelled_at = $3, updated_at = $4
        WHERE id = $5`,
        req.Reason, req.CancelledBy, now, now, bookingUUID)
    
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to cancel booking")
    }
    
    // Clear availability cache
    s.clearAvailabilityCache(booking.MasterID, booking.BookingDate)
    
    // TODO: Send cancellation notification
    
    return &pb.CancelBookingResponse{
        Message: "Booking cancelled successfully",
    }, nil
}

// Helper functions
func (s *BookingService) isTimeSlotAvailable(tx *sqlx.Tx, masterID uuid.UUID, date time.Time, timeSlot string) (bool, error) {
    var count int
    err := tx.Get(&count, `
        SELECT COUNT(*) FROM bookings 
        WHERE master_id = $1 AND booking_date = $2 AND booking_time = $3 AND status != 'cancelled'`,
        masterID, date, timeSlot)
    
    if err != nil {
        return false, err
    }
    
    return count == 0, nil
}

func (s *BookingService) getAvailableTimeSlots(masterID uuid.UUID, date time.Time) ([]TimeSlot, error) {
    db := database.GetDB()
    
    // Get existing bookings for the day
    var bookings []string
    err := db.Select(&bookings, `
        SELECT booking_time FROM bookings 
        WHERE master_id = $1 AND booking_date = $2 AND status != 'cancelled'`,
        masterID, date)
    
    if err != nil {
        return nil, err
    }
    
    // Generate time slots (9:00 to 18:00, 30-minute intervals)
    slots := []TimeSlot{}
    start := 9 * 60  // 9:00 AM in minutes
    end := 18 * 60   // 6:00 PM in minutes
    interval := 30   // 30 minutes
    
    for minutes := start; minutes < end; minutes += interval {
        timeStr := fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
        
        // Check if this slot is booked
        isAvailable := true
        for _, bookedTime := range bookings {
            if bookedTime == timeStr {
                isAvailable = false
                break
            }
        }
        
        slots = append(slots, TimeSlot{
            StartTime:   timeStr,
            EndTime:     fmt.Sprintf("%02d:%02d", (minutes+interval)/60, (minutes+interval)%60),
            IsAvailable: isAvailable,
        })
    }
    
    return slots, nil
}

func (s *BookingService) clearAvailabilityCache(masterID uuid.UUID, date time.Time) {
    cacheKey := fmt.Sprintf("availability:%s:%s", masterID.String(), date.Format("2006-01-02"))
    cache.Delete(context.Background(), cacheKey)
}

func generateConfirmationCode() string {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    code := make([]byte, 6)
    for i := range code {
        code[i] = charset[rand.Intn(len(charset))]
    }
    return string(code)
}

func isValidTimeFormat(timeStr string) bool {
    parts := strings.Split(timeStr, ":")
    if len(parts) != 2 {
        return false
    }
    
    hour, err1 := strconv.Atoi(parts[0])
    minute, err2 := strconv.Atoi(parts[1])
    
    return err1 == nil && err2 == nil && hour >= 0 && hour < 24 && minute >= 0 && minute < 60
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
    var cancelledAt string
    if booking.CancelledAt != nil {
        cancelledAt = booking.CancelledAt.Format(time.RFC3339)
    }
    
    var clientNotes, masterNotes, cancellationReason, cancelledBy, confirmationCode string
    if booking.ClientNotes != nil {
        clientNotes = *booking.ClientNotes
    }
    if booking.MasterNotes != nil {
        masterNotes = *booking.MasterNotes
    }
    if booking.CancellationReason != nil {
        cancellationReason = *booking.CancellationReason
    }
    if booking.CancelledBy != nil {
        cancelledBy = *booking.CancelledBy
    }
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
        protoServices = append(protoServices, &pb.Service{
            Id:              service.ID.String(),
            TenantId:        service.TenantID.String(),
            Category:        service.Category,
            Name:            service.Name,
            Description:     service.Description,
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