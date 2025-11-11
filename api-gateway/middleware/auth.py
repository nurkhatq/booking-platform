from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from typing import Optional, Dict
import logging

from shared.auth import decode_token
from shared.models import UserRole

logger = logging.getLogger(__name__)

security = HTTPBearer()


async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(security)
) -> Dict:
    """
    Dependency to get current authenticated user from JWT token.

    Returns user data from token payload.
    Raises HTTPException if token is invalid.
    """
    token = credentials.credentials

    payload = decode_token(token)
    if not payload:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid authentication credentials",
            headers={"WWW-Authenticate": "Bearer"},
        )

    # Check token type
    if payload.get("type") != "access":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token type",
            headers={"WWW-Authenticate": "Bearer"},
        )

    return payload


async def get_current_active_user(
    current_user: Dict = Depends(get_current_user)
) -> Dict:
    """Get current active user."""
    # Add additional checks if needed (e.g., check if user is active in DB)
    return current_user


def require_role(*allowed_roles: UserRole):
    """
    Dependency factory to require specific roles.

    Usage:
        @app.get("/admin")
        async def admin_endpoint(user = Depends(require_role(UserRole.SUPER_ADMIN))):
            ...
    """

    async def role_checker(current_user: Dict = Depends(get_current_user)) -> Dict:
        user_role = current_user.get("role")

        if user_role not in [role.value for role in allowed_roles]:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="Insufficient permissions"
            )

        return current_user

    return role_checker


async def get_optional_user(
    credentials: Optional[HTTPAuthorizationCredentials] = Depends(HTTPBearer(auto_error=False))
) -> Optional[Dict]:
    """
    Optional authentication - returns user if token provided, None otherwise.

    Useful for endpoints that behave differently for authenticated users.
    """
    if not credentials:
        return None

    try:
        token = credentials.credentials
        payload = decode_token(token)
        return payload
    except Exception as e:
        logger.warning(f"Optional auth failed: {e}")
        return None
