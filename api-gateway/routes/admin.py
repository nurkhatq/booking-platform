from fastapi import APIRouter, HTTPException, status, Depends
from pydantic import BaseModel
from typing import Optional
import httpx
import logging

from shared.config import settings
from shared.models import UserRole
from middleware.auth import require_role

logger = logging.getLogger(__name__)

router = APIRouter()

# Admin service URL
ADMIN_SERVICE_URL = f"http://admin-service:{settings.ADMIN_SERVICE_PORT if hasattr(settings, 'ADMIN_SERVICE_PORT') else 8005}"


@router.get("/tenants")
async def get_pending_tenants(
    current_user: dict = Depends(require_role(UserRole.SUPER_ADMIN))
):
    """
    Get all pending tenant applications.

    Only accessible by SUPER_ADMIN.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{ADMIN_SERVICE_URL}/tenants/pending",
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Admin service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to admin service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Admin service unavailable"
        )


@router.put("/tenant/{tenant_id}/approve")
async def approve_tenant(
    tenant_id: int,
    current_user: dict = Depends(require_role(UserRole.SUPER_ADMIN))
):
    """
    Approve tenant application.

    Only accessible by SUPER_ADMIN.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{ADMIN_SERVICE_URL}/tenant/{tenant_id}/approve",
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Tenant not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Admin service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to admin service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Admin service unavailable"
        )


@router.put("/tenant/{tenant_id}/reject")
async def reject_tenant(
    tenant_id: int,
    current_user: dict = Depends(require_role(UserRole.SUPER_ADMIN))
):
    """
    Reject tenant application.

    Only accessible by SUPER_ADMIN.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{ADMIN_SERVICE_URL}/tenant/{tenant_id}/reject",
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Tenant not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Admin service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to admin service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Admin service unavailable"
        )


@router.get("/statistics")
async def get_statistics(
    current_user: dict = Depends(require_role(UserRole.SUPER_ADMIN))
):
    """
    Get platform statistics.

    Only accessible by SUPER_ADMIN.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{ADMIN_SERVICE_URL}/statistics",
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Admin service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to admin service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Admin service unavailable"
        )
