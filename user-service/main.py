from fastapi import FastAPI, HTTPException, status, Depends
from pydantic import BaseModel, EmailStr
from sqlalchemy.orm import Session
from datetime import datetime, timedelta
import logging

from shared.config import settings
from shared.database import get_db, init_db, check_db_connection
from shared.models import User, Tenant, Location, UserRole, TenantStatus
from shared.auth import verify_password, get_password_hash, create_token_pair
from services.user_service import UserService

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="User Service",
    description="User authentication and management service",
    version="2.0.0"
)


# Request/Response models
class RegisterRequest(BaseModel):
    email: EmailStr
    password: str
    full_name: str
    phone: str
    business_name: str
    subdomain: str


class LoginRequest(BaseModel):
    email: EmailStr
    password: str


class RefreshTokenRequest(BaseModel):
    refresh_token: str


class ChangePasswordRequest(BaseModel):
    user_id: int
    old_password: str
    new_password: str


@app.on_event("startup")
async def startup_event():
    """Initialize database on startup."""
    logger.info("Starting User Service...")
    if check_db_connection():
        logger.info("Database connection successful")
    else:
        logger.error("Database connection failed")


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    db_healthy = check_db_connection()
    return {
        "status": "healthy" if db_healthy else "unhealthy",
        "service": "user-service",
        "database": "connected" if db_healthy else "disconnected"
    }


@app.post("/register", status_code=status.HTTP_201_CREATED)
async def register(data: RegisterRequest, db: Session = Depends(get_db)):
    """
    Register new business owner and create tenant.

    Creates:
    - New tenant with PENDING status
    - New user with OWNER role
    - Default location for the business
    """
    user_service = UserService(db)

    # Check if email already exists
    existing_user = db.query(User).filter(User.email == data.email).first()
    if existing_user:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Email already registered"
        )

    # Check if subdomain already exists
    existing_tenant = db.query(Tenant).filter(Tenant.subdomain == data.subdomain).first()
    if existing_tenant:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Subdomain already taken"
        )

    try:
        # Create tenant
        tenant = Tenant(
            subdomain=data.subdomain,
            business_name=data.business_name,
            phone=data.phone,
            email=data.email,
            status=TenantStatus.TRIAL,
            trial_end_date=datetime.utcnow() + timedelta(days=settings.DEFAULT_TRIAL_DAYS)
        )
        db.add(tenant)
        db.flush()

        # Create main location
        location = Location(
            tenant_id=tenant.id,
            name=f"{data.business_name} - Main",
            phone=data.phone,
            is_main=True
        )
        db.add(location)

        # Create owner user
        hashed_password = get_password_hash(data.password)
        user = User(
            tenant_id=tenant.id,
            email=data.email,
            phone=data.phone,
            password_hash=hashed_password,
            full_name=data.full_name,
            role=UserRole.OWNER,
            is_active=True
        )
        db.add(user)
        db.commit()

        logger.info(f"New tenant registered: {data.subdomain}")

        # Create tokens
        tokens = create_token_pair(user.id, user.email, user.role.value, tenant.id)

        return {
            "message": "Registration successful",
            "user_id": user.id,
            "tenant_id": tenant.id,
            "subdomain": tenant.subdomain,
            **tokens
        }

    except Exception as e:
        db.rollback()
        logger.error(f"Registration failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Registration failed"
        )


@app.post("/login")
async def login(data: LoginRequest, db: Session = Depends(get_db)):
    """
    Authenticate user and return JWT tokens.
    """
    # Find user by email
    user = db.query(User).filter(User.email == data.email).first()

    if not user or not verify_password(data.password, user.password_hash):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid email or password"
        )

    if not user.is_active:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Account is inactive"
        )

    # Create tokens
    tokens = create_token_pair(user.id, user.email, user.role.value, user.tenant_id)

    logger.info(f"User logged in: {user.email}")

    return {
        "message": "Login successful",
        "user": {
            "id": user.id,
            "email": user.email,
            "full_name": user.full_name,
            "role": user.role.value,
            "tenant_id": user.tenant_id
        },
        **tokens
    }


@app.post("/refresh-token")
async def refresh_token(data: RefreshTokenRequest):
    """
    Refresh access token using refresh token.
    """
    from shared.auth import decode_token, create_access_token

    payload = decode_token(data.refresh_token)

    if not payload or payload.get("type") != "refresh":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid refresh token"
        )

    # Create new access token
    token_data = {
        "sub": payload.get("sub"),
        "email": payload.get("email"),
        "role": payload.get("role"),
    }

    if payload.get("tenant_id"):
        token_data["tenant_id"] = payload.get("tenant_id")

    access_token = create_access_token(token_data)

    return {
        "access_token": access_token,
        "token_type": "bearer"
    }


@app.post("/change-password")
async def change_password(data: ChangePasswordRequest, db: Session = Depends(get_db)):
    """
    Change user password.
    """
    user = db.query(User).filter(User.id == data.user_id).first()

    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )

    if not verify_password(data.old_password, user.password_hash):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid old password"
        )

    # Update password
    user.password_hash = get_password_hash(data.new_password)
    db.commit()

    logger.info(f"Password changed for user: {user.email}")

    return {"message": "Password changed successfully"}


@app.get("/user/{user_id}")
async def get_user(user_id: int, db: Session = Depends(get_db)):
    """
    Get user by ID.
    """
    user = db.query(User).filter(User.id == user_id).first()

    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )

    return {
        "id": user.id,
        "email": user.email,
        "full_name": user.full_name,
        "phone": user.phone,
        "role": user.role.value,
        "tenant_id": user.tenant_id,
        "is_active": user.is_active,
        "created_at": user.created_at.isoformat()
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host=settings.USER_SERVICE_HOST if hasattr(settings, 'USER_SERVICE_HOST') else "0.0.0.0",
        port=settings.USER_SERVICE_PORT if hasattr(settings, 'USER_SERVICE_PORT') else 8001,
        reload=settings.DEBUG
    )
