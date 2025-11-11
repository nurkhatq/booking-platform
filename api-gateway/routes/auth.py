from fastapi import APIRouter, HTTPException, status, Depends
from pydantic import BaseModel, EmailStr, validator
from typing import Optional
import httpx
import logging

from shared.config import settings
from middleware.auth import get_current_user

logger = logging.getLogger(__name__)

router = APIRouter()

# User service URL
USER_SERVICE_URL = f"http://user-service:{settings.USER_SERVICE_PORT if hasattr(settings, 'USER_SERVICE_PORT') else 8001}"


# Request/Response models
class RegisterRequest(BaseModel):
    email: EmailStr
    password: str
    full_name: str
    phone: str
    business_name: str
    subdomain: str

    @validator('password')
    def password_strength(cls, v):
        if len(v) < 8:
            raise ValueError('Password must be at least 8 characters')
        return v

    @validator('subdomain')
    def subdomain_valid(cls, v):
        if not v.isalnum():
            raise ValueError('Subdomain must contain only letters and numbers')
        return v.lower()


class LoginRequest(BaseModel):
    email: EmailStr
    password: str


class RefreshTokenRequest(BaseModel):
    refresh_token: str


class ChangePasswordRequest(BaseModel):
    old_password: str
    new_password: str

    @validator('new_password')
    def password_strength(cls, v):
        if len(v) < 8:
            raise ValueError('Password must be at least 8 characters')
        return v


@router.post("/register", status_code=status.HTTP_201_CREATED)
async def register(data: RegisterRequest):
    """
    Register new business owner account.

    Creates new tenant and owner user.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{USER_SERVICE_URL}/register",
                json=data.dict(),
                timeout=10.0
            )

            if response.status_code == 201:
                return response.json()
            elif response.status_code == 400:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=response.json().get("detail", "Registration failed")
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="User service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to user service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="User service unavailable"
        )


@router.post("/login")
async def login(data: LoginRequest):
    """
    Login user and get JWT tokens.

    Returns access token and refresh token.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{USER_SERVICE_URL}/login",
                json=data.dict(),
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 401:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Invalid email or password"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="User service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to user service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="User service unavailable"
        )


@router.post("/refresh-token")
async def refresh_token(data: RefreshTokenRequest):
    """
    Refresh access token using refresh token.

    Returns new access token.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{USER_SERVICE_URL}/refresh-token",
                json=data.dict(),
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 401:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Invalid refresh token"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="User service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to user service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="User service unavailable"
        )


@router.post("/logout")
async def logout(current_user: dict = Depends(get_current_user)):
    """
    Logout current user.

    In a stateless JWT system, actual logout is handled client-side.
    This endpoint can be used for logging/auditing.
    """
    return {"message": "Logged out successfully"}


@router.get("/me")
async def get_current_user_info(current_user: dict = Depends(get_current_user)):
    """
    Get current authenticated user information.

    Returns user data from JWT token.
    """
    return {
        "user_id": current_user.get("sub"),
        "email": current_user.get("email"),
        "role": current_user.get("role"),
        "tenant_id": current_user.get("tenant_id")
    }


@router.post("/change-password")
async def change_password(
    data: ChangePasswordRequest,
    current_user: dict = Depends(get_current_user)
):
    """
    Change password for current user.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{USER_SERVICE_URL}/change-password",
                json={
                    "user_id": current_user.get("sub"),
                    "old_password": data.old_password,
                    "new_password": data.new_password
                },
                timeout=10.0
            )

            if response.status_code == 200:
                return {"message": "Password changed successfully"}
            elif response.status_code == 400:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="Invalid old password"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="User service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to user service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="User service unavailable"
        )
