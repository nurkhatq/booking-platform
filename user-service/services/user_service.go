package services

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "golang.org/x/crypto/bcrypt"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    
    "booking-platform/shared/auth"
    "booking-platform/shared/config"
    "booking-platform/shared/database"
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
    
    // Get user by email
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
    
    // Check password
    if user.PasswordHash == nil {
        return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
    }
    
    err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
    }
    
    // Generate JWT token
    token, err := auth.GenerateToken(user.ID.String(), string(user.Role), user.TenantID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to generate token")
    }
    
    // Generate refresh token
    refreshToken, err := auth.GenerateRefreshToken(user.ID.String())
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to generate refresh token")
    }
    
    // Update last login
    _, err = db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = $1", user.ID)
    if err != nil {
        // Log error but don't fail the login
        fmt.Printf("Failed to update last login: %v\n", err)
    }
    
    return &pb.LoginResponse{
        Token:        token,
        RefreshToken: refreshToken,
        User:         userToProto(&user),
    }, nil
}

func (s *UserService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
    // Validate refresh token
    claims, err := auth.ValidateToken(req.RefreshToken)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "Invalid refresh token")
    }
    
    db := database.GetDB()
    
    // Verify user still exists and is active
    var user models.User
    err = db.Get(&user, 
        "SELECT * FROM users WHERE id = $1 AND is_active = true", 
        claims.UserID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.Unauthenticated, "User not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Generate new JWT token
    token, err := auth.GenerateToken(user.ID.String(), string(user.Role), user.TenantID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to generate token")
    }
    
    return &pb.RefreshTokenResponse{
        Token: token,
    }, nil
}

func (s *UserService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
    // Validate token to get user info
    claims, err := auth.ValidateToken(req.Token)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "Invalid token")
    }
    
    // In a real implementation, you would:
    // 1. Add the token to a blacklist
    // 2. Remove refresh tokens for this user
    // 3. Log the logout event
    
    // For now, we'll just return success
    return &pb.LogoutResponse{
        Message: "Logged out successfully",
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
    var subdomainExists bool
    err = tx.Get(&subdomainExists, "SELECT EXISTS(SELECT 1 FROM tenants WHERE subdomain = $1)", req.Subdomain)
    if err != nil {
        return nil, status.Error(codes.Internal, "Database error")
    }
    if subdomainExists {
        return nil, status.Error(codes.AlreadyExists, "Subdomain already taken")
    }
    
    // Check if email is already registered
    var emailExists bool
    err = tx.Get(&emailExists, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.OwnerEmail)
    if err != nil {
        return nil, status.Error(codes.Internal, "Database error")
    }
    if emailExists {
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
    
    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.OwnerPassword), bcrypt.DefaultCost)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to hash password")
    }
    
    // Create owner user
    ownerID := uuid.New()
    _, err = tx.Exec(`
        INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, role, is_active, email_verified)
        VALUES ($1, $2, $3, $4, $5, $6, 'OWNER', true, false)`,
        ownerID, tenantID, req.OwnerEmail, string(hashedPassword), req.OwnerName, req.OwnerName)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to create owner user")
    }
    
    // Update tenant with owner ID
    _, err = tx.Exec("UPDATE tenants SET owner_id = $1 WHERE id = $2", ownerID, tenantID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to update tenant")
    }
    
    err = tx.Commit()
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to commit transaction")
    }
    
    return &pb.RegisterBusinessResponse{
        TenantId: tenantID.String(),
        Message:  "Business registration submitted for approval",
    }, nil
}

func (s *UserService) GetBusinessInfo(ctx context.Context, req *pb.GetBusinessInfoRequest) (*pb.GetBusinessInfoResponse, error) {
    db := database.GetDB()
    
    var tenant models.Tenant
    err := db.Get(&tenant, 
        "SELECT * FROM tenants WHERE subdomain = $1", 
        req.Subdomain)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Business not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    return &pb.GetBusinessInfoResponse{
        BusinessName: tenant.BusinessName,
        BusinessType: string(tenant.BusinessType),
        Subdomain:    tenant.Subdomain,
        Timezone:     tenant.Timezone,
        Status:       string(tenant.Status),
    }, nil
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

func (s *UserService) CreateClientSession(ctx context.Context, req *pb.CreateClientSessionRequest) (*pb.CreateClientSessionResponse, error) {
    db := database.GetDB()
    
    // Generate verification code
    verificationCode := fmt.Sprintf("%06d", time.Now().Unix()%1000000)
    expiresAt := time.Now().Add(10 * time.Minute)
    
    // Create or update client session
    sessionID := uuid.New()
    _, err := db.Exec(`
        INSERT INTO client_sessions (id, email, phone, first_name, last_name, verification_code, verification_expires, is_verified)
        VALUES ($1, $2, $3, $4, $5, $6, $7, false)
        ON CONFLICT (email) DO UPDATE SET
            phone = $3,
            first_name = $4,
            last_name = $5,
            verification_code = $6,
            verification_expires = $7,
            is_verified = false,
            updated_at = CURRENT_TIMESTAMP`,
        sessionID, req.Email, req.Phone, req.FirstName, req.LastName, verificationCode, expiresAt)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to create client session")
    }
    
    // In a real implementation, you would send the verification code via SMS/Email
    // For now, we'll just return it in the response (for testing)
    
    return &pb.CreateClientSessionResponse{
        SessionId: sessionID.String(),
        Message:   fmt.Sprintf("Verification code sent. Code: %s (for testing)", verificationCode),
    }, nil
}

func (s *UserService) VerifyClientCode(ctx context.Context, req *pb.VerifyClientCodeRequest) (*pb.VerifyClientCodeResponse, error) {
    db := database.GetDB()
    
    sessionID, err := uuid.Parse(req.SessionId)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, "Invalid session ID")
    }
    
    // Get client session
    var session models.ClientSession
    err = db.Get(&session, 
        "SELECT * FROM client_sessions WHERE id = $1", 
        sessionID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "Session not found")
        }
        return nil, status.Error(codes.Internal, "Database error")
    }
    
    // Check if code is correct and not expired
    if session.VerificationCode == nil || *session.VerificationCode != req.Code {
        return nil, status.Error(codes.InvalidArgument, "Invalid verification code")
    }
    
    if session.VerificationExpires == nil || time.Now().After(*session.VerificationExpires) {
        return nil, status.Error(codes.DeadlineExceeded, "Verification code expired")
    }
    
    // Generate session token
    sessionToken := uuid.New().String()
    sessionExpires := time.Now().Add(24 * time.Hour)
    
    // Update session
    _, err = db.Exec(`
        UPDATE client_sessions 
        SET is_verified = true, session_token = $1, session_expires = $2, last_used = CURRENT_TIMESTAMP
        WHERE id = $3`,
        sessionToken, sessionExpires, sessionID)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to update session")
    }
    
    // Generate JWT token for client
    token, err := auth.GenerateClientToken(sessionID.String())
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to generate token")
    }
    
    return &pb.VerifyClientCodeResponse{
        Token:   token,
        Message: "Client verified successfully",
    }, nil
}

func userToProto(user *models.User) *pb.User {
    var firstName string
    if user.FirstName != nil {
        firstName = *user.FirstName
    }
    
    var lastName string
    if user.LastName != nil {
        lastName = *user.LastName
    }
    
    var phone string
    if user.Phone != nil {
        phone = *user.Phone
    }
    
    var tenantID string
    if user.TenantID != nil {
        tenantID = user.TenantID.String()
    }
    
    var locationID string
    if user.LocationID != nil {
        locationID = user.LocationID.String()
    }
    
    var lastLogin string
    if user.LastLogin != nil {
        lastLogin = user.LastLogin.Format(time.RFC3339)
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
        LastLogin:     lastLogin,
        CreatedAt:     user.CreatedAt.Format(time.RFC3339),
        UpdatedAt:     user.UpdatedAt.Format(time.RFC3339),
    }
}