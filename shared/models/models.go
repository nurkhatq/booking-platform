package models

import (
    "time"
    "github.com/google/uuid"
)

type UserRole string

const (
    RoleSuperAdmin UserRole = "SUPER_ADMIN"
    RoleOwner      UserRole = "OWNER"
    RoleManager    UserRole = "MANAGER"
    RoleMaster     UserRole = "MASTER"
    RoleClient     UserRole = "CLIENT"
)

type BusinessType string

const (
    BusinessBarbershop   BusinessType = "barbershop"
    BusinessSalon        BusinessType = "salon"
    BusinessClinic       BusinessType = "clinic"
    BusinessSpa          BusinessType = "spa"
    BusinessBeautyCenter BusinessType = "beauty_center"
)

type BookingStatus string

const (
    BookingPending   BookingStatus = "pending"
    BookingConfirmed BookingStatus = "confirmed"
    BookingCompleted BookingStatus = "completed"
    BookingCancelled BookingStatus = "cancelled"
)

type TenantStatus string

const (
    TenantPending   TenantStatus = "pending"
    TenantApproved  TenantStatus = "approved"
    TenantActive    TenantStatus = "active"
    TenantSuspended TenantStatus = "suspended"
    TenantExpired   TenantStatus = "expired"
)

type Tenant struct {
    ID                  uuid.UUID     `json:"id" db:"id"`
    BusinessName        string        `json:"business_name" db:"business_name"`
    BusinessType        BusinessType  `json:"business_type" db:"business_type"`
    Subdomain          string        `json:"subdomain" db:"subdomain"`
    Timezone           string        `json:"timezone" db:"timezone"`
    OwnerID            *uuid.UUID    `json:"owner_id" db:"owner_id"`
    Status             TenantStatus  `json:"status" db:"status"`
    TrialStartDate     *time.Time    `json:"trial_start_date" db:"trial_start_date"`
    TrialEndDate       *time.Time    `json:"trial_end_date" db:"trial_end_date"`
    SubscriptionEndDate *time.Time   `json:"subscription_end_date" db:"subscription_end_date"`
    Settings           map[string]interface{} `json:"settings" db:"settings"`
    Branding           map[string]interface{} `json:"branding" db:"branding"`
    CreatedAt          time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time     `json:"updated_at" db:"updated_at"`
}

type Location struct {
    ID           uuid.UUID                  `json:"id" db:"id"`
    TenantID     uuid.UUID                  `json:"tenant_id" db:"tenant_id"`
    Name         string                     `json:"name" db:"name"`
    Address      string                     `json:"address" db:"address"`
    City         string                     `json:"city" db:"city"`
    Country      string                     `json:"country" db:"country"`
    Phone        *string                    `json:"phone" db:"phone"`
    ManagerID    *uuid.UUID                 `json:"manager_id" db:"manager_id"`
    WorkingHours map[string]interface{}     `json:"working_hours" db:"working_hours"`
    Settings     map[string]interface{}     `json:"settings" db:"settings"`
    IsActive     bool                       `json:"is_active" db:"is_active"`
    CreatedAt    time.Time                  `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time                  `json:"updated_at" db:"updated_at"`
}

type User struct {
    ID            uuid.UUID                  `json:"id" db:"id"`
    TenantID      *uuid.UUID                 `json:"tenant_id" db:"tenant_id"`
    LocationID    *uuid.UUID                 `json:"location_id" db:"location_id"`
    Email         string                     `json:"email" db:"email"`
    Phone         *string                    `json:"phone" db:"phone"`
    PasswordHash  *string                    `json:"-" db:"password_hash"`
    FirstName     *string                    `json:"first_name" db:"first_name"`
    LastName      *string                    `json:"last_name" db:"last_name"`
    Role          UserRole                   `json:"role" db:"role"`
    IsActive      bool                       `json:"is_active" db:"is_active"`
    EmailVerified bool                       `json:"email_verified" db:"email_verified"`
    PhoneVerified bool                       `json:"phone_verified" db:"phone_verified"`
    LastLogin     *time.Time                 `json:"last_login" db:"last_login"`
    Settings      map[string]interface{}     `json:"settings" db:"settings"`
    CreatedAt     time.Time                  `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time                  `json:"updated_at" db:"updated_at"`
}

type Master struct {
    ID                   uuid.UUID                  `json:"id" db:"id"`
    UserID               uuid.UUID                  `json:"user_id" db:"user_id"`
    TenantID             uuid.UUID                  `json:"tenant_id" db:"tenant_id"`
    LocationID           uuid.UUID                  `json:"location_id" db:"location_id"`
    Bio                  *string                    `json:"bio" db:"bio"`
    PhotoURL             *string                    `json:"photo_url" db:"photo_url"`
    Specialization       *string                    `json:"specialization" db:"specialization"`
    ExperienceYears      int                        `json:"experience_years" db:"experience_years"`
    Rating               float64                    `json:"rating" db:"rating"`
    TotalReviews         int                        `json:"total_reviews" db:"total_reviews"`
    Permissions          map[string]interface{}     `json:"permissions" db:"permissions"`
    Availability         map[string]interface{}     `json:"availability" db:"availability"`
    IsVisible            bool                       `json:"is_visible" db:"is_visible"`
    IsAcceptingBookings  bool                       `json:"is_accepting_bookings" db:"is_accepting_bookings"`
    CreatedAt            time.Time                  `json:"created_at" db:"created_at"`
    UpdatedAt            time.Time                  `json:"updated_at" db:"updated_at"`
    
    // Joined fields
    User *User `json:"user,omitempty"`
}

type Service struct {
    ID              uuid.UUID                  `json:"id" db:"id"`
    TenantID        uuid.UUID                  `json:"tenant_id" db:"tenant_id"`
    Category        string                     `json:"category" db:"category"`
    Name            string                     `json:"name" db:"name"`
    Description     *string                    `json:"description" db:"description"`
    BasePrice       float64                    `json:"base_price" db:"base_price"`
    BaseDuration    int                        `json:"base_duration" db:"base_duration"`
    IsActive        bool                       `json:"is_active" db:"is_active"`
    PopularityScore int                        `json:"popularity_score" db:"popularity_score"`
    Settings        map[string]interface{}     `json:"settings" db:"settings"`
    CreatedAt       time.Time                  `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time                  `json:"updated_at" db:"updated_at"`
}

type Booking struct {
    ID                 uuid.UUID     `json:"id" db:"id"`
    TenantID           uuid.UUID     `json:"tenant_id" db:"tenant_id"`
    LocationID         uuid.UUID     `json:"location_id" db:"location_id"`
    MasterID           uuid.UUID     `json:"master_id" db:"master_id"`
    ServiceID          uuid.UUID     `json:"service_id" db:"service_id"`
    ClientSessionID    uuid.UUID     `json:"client_session_id" db:"client_session_id"`
    ClientName         string        `json:"client_name" db:"client_name"`
    ClientEmail        string        `json:"client_email" db:"client_email"`
    ClientPhone        string        `json:"client_phone" db:"client_phone"`
    BookingDate        time.Time     `json:"booking_date" db:"booking_date"`
    BookingTime        string        `json:"booking_time" db:"booking_time"`
    Duration           int           `json:"duration" db:"duration"`
    Price              float64       `json:"price" db:"price"`
    Status             BookingStatus `json:"status" db:"status"`
    ClientNotes        *string       `json:"client_notes" db:"client_notes"`
    MasterNotes        *string       `json:"master_notes" db:"master_notes"`
    CancellationReason *string       `json:"cancellation_reason" db:"cancellation_reason"`
    CancelledBy        *string       `json:"cancelled_by" db:"cancelled_by"`
    CancelledAt        *time.Time    `json:"cancelled_at" db:"cancelled_at"`
    CreatedBy          string        `json:"created_by" db:"created_by"`
    ConfirmationCode   *string       `json:"confirmation_code" db:"confirmation_code"`
    ReminderSent24h    bool          `json:"reminder_sent_24h" db:"reminder_sent_24h"`
    ReminderSent2h     bool          `json:"reminder_sent_2h" db:"reminder_sent_2h"`
    CreatedAt          time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time     `json:"updated_at" db:"updated_at"`
    
    // Joined fields
    Master   *Master   `json:"master,omitempty"`
    Service  *Service  `json:"service,omitempty"`
    Location *Location `json:"location,omitempty"`
}

type ClientSession struct {
    ID                 uuid.UUID                  `json:"id" db:"id"`
    Email              string                     `json:"email" db:"email"`
    Phone              *string                    `json:"phone" db:"phone"`
    FirstName          *string                    `json:"first_name" db:"first_name"`
    LastName           *string                    `json:"last_name" db:"last_name"`
    VerificationCode   *string                    `json:"-" db:"verification_code"`
    VerificationExpires *time.Time                `json:"-" db:"verification_expires"`
    IsVerified         bool                       `json:"is_verified" db:"is_verified"`
    SessionToken       *string                    `json:"-" db:"session_token"`
    SessionExpires     *time.Time                 `json:"session_expires" db:"session_expires"`
    LastUsed           *time.Time                 `json:"last_used" db:"last_used"`
    Preferences        map[string]interface{}     `json:"preferences" db:"preferences"`
    CreatedAt          time.Time                  `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time                  `json:"updated_at" db:"updated_at"`
}

// API Request/Response structs
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
    Token        string `json:"token"`
    RefreshToken string `json:"refresh_token"`
    User         User   `json:"user"`
}

type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
    Token        string `json:"token"`
    RefreshToken string `json:"refresh_token"`
}

type TenantRegistrationRequest struct {
    BusinessName string       `json:"business_name" binding:"required"`
    BusinessType BusinessType `json:"business_type" binding:"required"`
    Subdomain    string       `json:"subdomain" binding:"required"`
    Timezone     string       `json:"timezone" binding:"required"`
    OwnerEmail   string       `json:"owner_email" binding:"required,email"`
    OwnerPhone   string       `json:"owner_phone" binding:"required"`
    OwnerName    string       `json:"owner_name" binding:"required"`
}

type BookingRequest struct {
    MasterID    uuid.UUID `json:"master_id" binding:"required"`
    ServiceID   uuid.UUID `json:"service_id" binding:"required"`
    LocationID  uuid.UUID `json:"location_id" binding:"required"`
    BookingDate string    `json:"booking_date" binding:"required"`
    BookingTime string    `json:"booking_time" binding:"required"`
    ClientName  string    `json:"client_name" binding:"required"`
    ClientEmail string    `json:"client_email" binding:"required,email"`
    ClientPhone string    `json:"client_phone" binding:"required"`
    ClientNotes *string   `json:"client_notes"`
}

type ClientVerificationRequest struct {
    Email string `json:"email" binding:"required,email"`
    Phone string `json:"phone" binding:"required"`
    Name  string `json:"name" binding:"required"`
}

type ClientVerificationResponse struct {
    SessionID string `json:"session_id"`
    Message   string `json:"message"`
}

type VerifyCodeRequest struct {
    SessionID string `json:"session_id" binding:"required"`
    Code      string `json:"code" binding:"required"`
}

type VerifyCodeResponse struct {
    Token   string `json:"token"`
    Message string `json:"message"`
}

type TenantStats struct {
    TotalBookings         int32   `json:"total_bookings" db:"total_bookings"`
    CompletedBookings     int32   `json:"completed_bookings" db:"completed_bookings"`
    CancelledBookings     int32   `json:"cancelled_bookings" db:"cancelled_bookings"`
    TotalRevenue          float64 `json:"total_revenue" db:"total_revenue"`
    AvgBookingValue       float64 `json:"avg_booking_value" db:"avg_booking_value"`
    TotalClients          int32   `json:"total_clients" db:"total_clients"`
    ActiveMasters         int32   `json:"active_masters" db:"active_masters"`
    MostPopularService    string  `json:"most_popular_service" db:"most_popular_service"`
    CustomerRetentionRate float64 `json:"customer_retention_rate" db:"customer_retention_rate"`
}