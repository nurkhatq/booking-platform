package services

import (
    "context"
    "crypto/rand"
    "database/sql"
    "fmt"
    "math/big"
    "strings"
    "time"
    
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    
    "booking-platform/shared/config"
    "booking-platform/shared/database"
    "booking-platform/shared/auth"
    "booking-platform/shared/models"
    pb "booking-platform/user-service/proto"
)

type UserService struct {
    pb.UnimplementedUserServiceServer
    config *config.Config
}

func NewUserService(cfg *config.Config) *UserService {
    return &UserService{
        config: cfg,
    }
}

func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
    db := database.GetDB()
    
    var user models.User
    err := db.Get(&user, 
        "SELECT * FROM users WHERE email = $1 AND is_active = true", 
        req.Email)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Verify password
    if user.PasswordHash == nil {
        return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
    }
    
    if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
        return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
    }
    
    // Generate tokens
    token, err := auth.GenerateToken(
        user.ID.String(),
        user.TenantID.String(),
        string(user.Role),
        user.Email,
        s.config.JWT.Expiry,
    )
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to generate token")
    }
    
    refreshToken, err := auth.GenerateToken(
        user.ID.String(),
        user.TenantID.String(),
        string(user.Role),
        user.Email,
        s.config.JWT.RefreshExpiry,
    )
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to generate refresh token")
    }
    
    // Update last login
    _, err = db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = $1", user.ID)
    if err != nil {
        // Log error but don't fail the login
        fmt.Printf("Failed to update last login for user %s: %v\n", user.ID, err)
    }
    
    return &pb.LoginResponse{
        Token:        token,
        RefreshToken: refreshToken,
        User:         userToProto(&user),
    }, nil
}

func (s *UserService) RegisterBusiness(ctx context.Context, req *pb.RegisterBusinessRequest) (*pb.RegisterBusinessResponse, error) {
    db := database.GetDB()
    
    // Start transaction
    tx, err := db.Beginx()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to start transaction")
    }
    defer tx.Rollback()
    
    // Check if subdomain is available
    var count int
    err = tx.Get(&count, "SELECT COUNT(*) FROM tenants WHERE subdomain = $1", req.Subdomain)
    if err != nil {
        return nil, status.Error(codes.Internal, "Database error")
    }
    if count > 0 {
        return nil, status.Error(codes.AlreadyExists, "Subdomain already taken")
    }
    
    // Check if email is already registered
    err = tx.Get(&count, "SELECT COUNT(*) FROM users WHERE email = $1", req.OwnerEmail)
    if err != nil {
        return nil, status.Error(codes.Internal, "Database error")
    }
    if count > 0 {
        return nil, status.Error(codes.AlreadyExists, "Email already registered")
    }
    
    // Create tenant
    tenantID := uuid.New()
    _, err = tx.Exec(`
        INSERT INTO tenants (id, business_name, business_type, subdomain, timezone, status)
        VALUES ($1, $2, $3, $4, $5, 'pending')`,
        tenantID, req.BusinessName, req.BusinessType, req.Subdomain, req.Timezone)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to create tenant")
    }
    
    // Create owner user
    userID := uuid.New()
    names := strings.Fields(req.OwnerName)
    firstName := names[0]
    var lastName string
    if len(names) > 1 {
        lastName = strings.Join(names[1:], " ")
    }
    
    _, err = tx.Exec(`
        INSERT INTO users (id, tenant_id, email, phone, first_name, last_name, role, is_active, email_verified)
        VALUES ($1, $2, $3, $4, $5, $6, 'OWNER', true, false)`,
        userID, tenantID, req.OwnerEmail, req.OwnerPhone, firstName, lastName)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to create owner user")
    }
    
    // Update tenant with owner_id
    _, err = tx.Exec("UPDATE tenants SET owner_id = $1 WHERE id = $2", userID, tenantID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to update tenant owner")
    }
    
    // Commit transaction
    if err = tx.Commit(); err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    // TODO: Send notification email to admin about new registration
    
    return &pb.RegisterBusinessResponse{
        TenantId: tenantID.String(),
        Message:  "Business registration submitted for approval",
    }, nil
}

func (s *UserService) CreateClientSession(ctx context.Context, req *pb.CreateClientSessionRequest) (*pb.CreateClientSessionResponse, error) {
    db := database.GetDB()
    
    // Check if client session already exists
    var existingSession models.ClientSession
    err := db.Get(&existingSession, "SELECT * FROM client_sessions WHERE email = $1", req.Email)
    
    sessionID := uuid.New()
    verificationCode := generateVerificationCode()
    expirationTime := time.Now().Add(10 * time.Minute) // 10 minutes expiry
    
    names := strings.Fields(req.Name)
    firstName := names[0]
    var lastName string
    if len(names) > 1 {
        lastName = strings.Join(names[1:], " ")
    }
    
    if err == sql.ErrNoRows {
        // Create new session
        _, err = db.Exec(`
            INSERT INTO client_sessions (id, email, phone, first_name, last_name, verification_code, verification_expires, is_verified)
            VALUES ($1, $2, $3, $4, $5, $6, $7, false)`,
            sessionID, req.Email, req.Phone, firstName, lastName, verificationCode, expirationTime)
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to create client session")
        }
    } else if err == nil {
        // Update existing session
        sessionID = existingSession.ID
        _, err = db.Exec(`
            UPDATE client_sessions 
            SET phone = $1, first_name = $2, last_name = $3, verification_code = $4, 
                verification_expires = $5, is_verified = false, updated_at = CURRENT_TIMESTAMP
            WHERE id = $6`,
            req.Phone, firstName, lastName, verificationCode, expirationTime, sessionID)
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to update client session")
        }
    } else {
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // TODO: Send verification code via SMS/Email
    fmt.Printf("Verification code for %s: %s\n", req.Email, verificationCode)
    
    return &pb.CreateClientSessionResponse{
        SessionId: sessionID.String(),
        Message:   "Verification code sent",
    }, nil
}

func (s *UserService) VerifyClientCode(ctx context.Context, req *pb.VerifyClientCodeRequest) (*pb.VerifyClientCodeResponse, error) {
    db := database.GetDB()
    
    sessionUUID, err := uuid.Parse(req.SessionId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid session ID")
    }
    
    var session models.ClientSession
    err = db.Get(&session, "SELECT * FROM client_sessions WHERE id = $1", sessionUUID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Session not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Check if verification code matches and hasn't expired
    if session.VerificationCode == nil || *session.VerificationCode != req.Code {
        return nil, status.Error(codes.InvalidArgument, "Invalid verification code")
    }
    
    if session.VerificationExpires != nil && session.VerificationExpires.Before(time.Now()) {
        return nil, status.Error(codes.DeadlineExceeded, "Verification code expired")
    }
    
    // Generate client token
    token, err := auth.GenerateClientToken(sessionUUID.String(), session.Email)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to generate token")
    }
    
    // Update session
    sessionExpires := time.Now().Add(30 * 24 * time.Hour) // 30 days
    _, err = db.Exec(`
        UPDATE client_sessions 
        SET is_verified = true, session_token = $1, session_expires = $2, 
            last_used = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
        WHERE id = $3`,
        token, sessionExpires, sessionUUID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to update session")
    }
    
    return &pb.VerifyClientCodeResponse{
        Token:   token,
        Message: "Verification successful",
    }, nil
}

func (s *UserService) GetBusinessInfo(ctx context.Context, req *pb.GetBusinessInfoRequest) (*pb.GetBusinessInfoResponse, error) {
    db := database.GetDB()
    
    // Get tenant by subdomain
    var tenant models.Tenant
    err := db.Get(&tenant, 
        "SELECT * FROM tenants WHERE subdomain = $1 AND status IN ('active', 'approved')", 
        req.Subdomain)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Business not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Get locations
    var locations []models.Location
    err = db.Select(&locations, 
        "SELECT * FROM locations WHERE tenant_id = $1 AND is_active = true ORDER BY name", 
        tenant.ID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to get locations")
    }
    
    // Convert to proto
    protoLocations := make([]*pb.Location, len(locations))
    for i, loc := range locations {
        protoLocations[i] = locationToProto(&loc)
    }
    
    return &pb.GetBusinessInfoResponse{
        BusinessName: tenant.BusinessName,
        BusinessType: string(tenant.BusinessType),
        Timezone:     tenant.Timezone,
        Locations:    protoLocations,
    }, nil
}

// Helper functions
func generateVerificationCode() string {
    const charset = "0123456789"
    code := make([]byte, 6)
    for i := range code {
        n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        code[i] = charset[n.Int64()]
    }
    return string(code)
}

func userToProto(user *models.User) *pb.User {
    var tenantID, locationID string
    if user.TenantID != nil {
        tenantID = user.TenantID.String()
    }
    if user.LocationID != nil {
        locationID = user.LocationID.String()
    }
    
    var firstName, lastName, phone string
    if user.FirstName != nil {
        firstName = *user.FirstName
    }
    if user.LastName != nil {
        lastName = *user.LastName
    }
    if user.Phone != nil {
        phone = *user.Phone
    }
    
    return &pb.User{
        Id:            user.ID.String(),
        TenantId:      tenantID,
        LocationId:    locationID,
        Email:         user.Email,
        Phone:         phone,
        FirstName:     firstName,
        LastName:      lastName,
        Role:          string(user.Role),
        IsActive:      user.IsActive,
        EmailVerified: user.EmailVerified,
        PhoneVerified: user.PhoneVerified,
        CreatedAt:     user.CreatedAt.Format(time.RFC3339),
        UpdatedAt:     user.UpdatedAt.Format(time.RFC3339),
    }
}

func locationToProto(location *models.Location) *pb.Location {
    var managerID, phone string
    if location.ManagerID != nil {
        managerID = location.ManagerID.String()
    }
    if location.Phone != nil {
        phone = *location.Phone
    }
    
    return &pb.Location{
        Id:        location.ID.String(),
        TenantId:  location.TenantID.String(),
        Name:      location.Name,
        Address:   location.Address,
        City:      location.City,
        Country:   location.Country,
        Phone:     phone,
        ManagerId: managerID,
        IsActive:  location.IsActive,
    }
}

// Placeholder implementations for other methods
func (s *UserService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) UpdateBusinessInfo(ctx context.Context, req *pb.UpdateBusinessInfoRequest) (*pb.UpdateBusinessInfoResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest) (*pb.GetUserProfileResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) UpdateUserProfile(ctx context.Context, req *pb.UpdateUserProfileRequest) (*pb.UpdateUserProfileResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) GetClientSession(ctx context.Context, req *pb.GetClientSessionRequest) (*pb.GetClientSessionResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) CreateLocation(ctx context.Context, req *pb.CreateLocationRequest) (*pb.CreateLocationResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) UpdateLocation(ctx context.Context, req *pb.UpdateLocationRequest) (*pb.UpdateLocationResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) GetLocations(ctx context.Context, req *pb.GetLocationsRequest) (*pb.GetLocationsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) CreateMaster(ctx context.Context, req *pb.CreateMasterRequest) (*pb.CreateMasterResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) UpdateMaster(ctx context.Context, req *pb.UpdateMasterRequest) (*pb.UpdateMasterResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *UserService) GetMasters(ctx context.Context, req *pb.GetMastersRequest) (*pb.GetMastersResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}
