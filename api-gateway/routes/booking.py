from fastapi import APIRouter, HTTPException, status, Depends, Query
from pydantic import BaseModel
from typing import Optional, List
from datetime import datetime, date
import httpx
import logging

from shared.config import settings
from middleware.auth import get_current_user, get_optional_user

logger = logging.getLogger(__name__)

router = APIRouter()

# Booking service URL
BOOKING_SERVICE_URL = f"http://booking-service:{settings.BOOKING_SERVICE_PORT if hasattr(settings, 'BOOKING_SERVICE_PORT') else 8002}"


# Request/Response models
class CreateBookingRequest(BaseModel):
    subdomain: str
    client_phone: str
    client_name: str
    master_id: int
    service_id: int
    booking_date: datetime
    notes: Optional[str] = None


class UpdateBookingRequest(BaseModel):
    booking_date: Optional[datetime] = None
    status: Optional[str] = None
    notes: Optional[str] = None


@router.get("/public/business/{subdomain}")
async def get_business_info(subdomain: str):
    """
    Get public business information by subdomain.

    Available to everyone without authentication.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{BOOKING_SERVICE_URL}/public/business/{subdomain}",
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Business not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )


@router.get("/public/business/{subdomain}/services")
async def get_business_services(subdomain: str):
    """
    Get all active services for a business.

    Public endpoint - no authentication required.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{BOOKING_SERVICE_URL}/public/business/{subdomain}/services",
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Business not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )


@router.get("/public/business/{subdomain}/masters")
async def get_business_masters(
    subdomain: str,
    service_id: Optional[int] = Query(None)
):
    """
    Get all active masters for a business.

    Optionally filter by service_id.
    Public endpoint - no authentication required.
    """
    try:
        params = {}
        if service_id:
            params["service_id"] = service_id

        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{BOOKING_SERVICE_URL}/public/business/{subdomain}/masters",
                params=params,
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Business not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )


@router.get("/public/business/{subdomain}/availability")
async def check_availability(
    subdomain: str,
    master_id: int = Query(...),
    date: date = Query(...)
):
    """
    Check master availability for a specific date.

    Returns available time slots.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{BOOKING_SERVICE_URL}/public/business/{subdomain}/availability",
                params={"master_id": master_id, "date": date.isoformat()},
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Business or master not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )


@router.post("/public/booking", status_code=status.HTTP_201_CREATED)
async def create_public_booking(data: CreateBookingRequest):
    """
    Create a new booking (public endpoint for clients).

    Sends WhatsApp confirmation to client.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{BOOKING_SERVICE_URL}/public/booking",
                json=data.dict(),
                timeout=15.0
            )

            if response.status_code == 201:
                return response.json()
            elif response.status_code == 400:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=response.json().get("detail", "Invalid booking data")
                )
            elif response.status_code == 409:
                raise HTTPException(
                    status_code=status.HTTP_409_CONFLICT,
                    detail="Time slot not available"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )


@router.get("/bookings")
async def get_bookings(
    current_user: dict = Depends(get_current_user),
    date: Optional[date] = Query(None),
    status: Optional[str] = Query(None)
):
    """
    Get bookings for current user.

    Filters based on user role:
    - OWNER: all tenant bookings
    - MANAGER: location bookings
    - MASTER: own bookings
    """
    try:
        params = {
            "user_id": current_user.get("sub"),
            "role": current_user.get("role"),
            "tenant_id": current_user.get("tenant_id")
        }

        if date:
            params["date"] = date.isoformat()
        if status:
            params["status"] = status

        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{BOOKING_SERVICE_URL}/bookings",
                params=params,
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )


@router.put("/booking/{booking_id}")
async def update_booking(
    booking_id: int,
    data: UpdateBookingRequest,
    current_user: dict = Depends(get_current_user)
):
    """
    Update booking.

    Requires appropriate permissions.
    """
    try:
        request_data = data.dict(exclude_unset=True)
        request_data.update({
            "user_id": current_user.get("sub"),
            "role": current_user.get("role")
        })

        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{BOOKING_SERVICE_URL}/booking/{booking_id}",
                json=request_data,
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 403:
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail="Insufficient permissions"
                )
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Booking not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )


@router.delete("/booking/{booking_id}")
async def cancel_booking(
    booking_id: int,
    current_user: dict = Depends(get_current_user)
):
    """
    Cancel booking.

    Sends WhatsApp notification to client.
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.delete(
                f"{BOOKING_SERVICE_URL}/booking/{booking_id}",
                params={
                    "user_id": current_user.get("sub"),
                    "role": current_user.get("role")
                },
                timeout=10.0
            )

            if response.status_code == 200:
                return response.json()
            elif response.status_code == 403:
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail="Insufficient permissions"
                )
            elif response.status_code == 404:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Booking not found"
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Booking service error"
                )

    except httpx.RequestError as e:
        logger.error(f"Failed to connect to booking service: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Booking service unavailable"
        )
