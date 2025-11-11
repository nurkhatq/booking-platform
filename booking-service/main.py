from fastapi import FastAPI, HTTPException, status, Depends, Query
from pydantic import BaseModel
from sqlalchemy.orm import Session
from datetime import datetime, date, time, timedelta
from typing import Optional, List
import httpx
import logging

from shared.config import settings
from shared.database import get_db, check_db_connection
from shared.models import (
    Tenant, Service, Master, Booking, Client, MasterSchedule,
    MasterService, BookingStatus, TenantStatus
)
from services.booking_service import BookingService

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="Booking Service",
    description="Booking management service",
    version="2.0.0"
)

# WhatsApp service URL
WHATSAPP_SERVICE_URL = settings.WHATSAPP_SERVICE_URL


# Request/Response models
class CreateBookingRequest(BaseModel):
    subdomain: str
    client_phone: str
    client_name: str
    master_id: int
    service_id: int
    booking_date: datetime
    notes: Optional[str] = None


@app.on_event("startup")
async def startup_event():
    """Initialize on startup."""
    logger.info("Starting Booking Service...")
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
        "service": "booking-service",
        "database": "connected" if db_healthy else "disconnected"
    }


@app.get("/public/business/{subdomain}")
async def get_business_info(subdomain: str, db: Session = Depends(get_db)):
    """
    Get public business information.
    """
    tenant = db.query(Tenant).filter(
        Tenant.subdomain == subdomain,
        Tenant.status.in_([TenantStatus.ACTIVE, TenantStatus.TRIAL])
    ).first()

    if not tenant:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Business not found"
        )

    return {
        "id": tenant.id,
        "business_name": tenant.business_name,
        "subdomain": tenant.subdomain,
        "phone": tenant.phone,
        "email": tenant.email,
        "description": tenant.description,
        "status": tenant.status.value
    }


@app.get("/public/business/{subdomain}/services")
async def get_business_services(subdomain: str, db: Session = Depends(get_db)):
    """
    Get all active services for a business.
    """
    tenant = db.query(Tenant).filter(Tenant.subdomain == subdomain).first()

    if not tenant:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Business not found"
        )

    services = db.query(Service).filter(
        Service.tenant_id == tenant.id,
        Service.is_active == True
    ).all()

    return {
        "services": [
            {
                "id": s.id,
                "name": s.name,
                "description": s.description,
                "duration_minutes": s.duration_minutes,
                "price": float(s.price)
            }
            for s in services
        ]
    }


@app.get("/public/business/{subdomain}/masters")
async def get_business_masters(
    subdomain: str,
    service_id: Optional[int] = Query(None),
    db: Session = Depends(get_db)
):
    """
    Get all active masters for a business.
    Optionally filter by service.
    """
    tenant = db.query(Tenant).filter(Tenant.subdomain == subdomain).first()

    if not tenant:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Business not found"
        )

    query = db.query(Master).filter(
        Master.tenant_id == tenant.id,
        Master.is_active == True
    )

    if service_id:
        # Filter masters who provide this service
        query = query.join(MasterService).filter(MasterService.service_id == service_id)

    masters = query.all()

    return {
        "masters": [
            {
                "id": m.id,
                "full_name": m.full_name,
                "description": m.description,
                "phone": m.phone
            }
            for m in masters
        ]
    }


@app.get("/public/business/{subdomain}/availability")
async def check_availability(
    subdomain: str,
    master_id: int = Query(...),
    date: date = Query(...),
    db: Session = Depends(get_db)
):
    """
    Check master availability for a specific date.
    Returns available time slots.
    """
    tenant = db.query(Tenant).filter(Tenant.subdomain == subdomain).first()

    if not tenant:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Business not found"
        )

    master = db.query(Master).filter(
        Master.id == master_id,
        Master.tenant_id == tenant.id
    ).first()

    if not master:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Master not found"
        )

    booking_service = BookingService(db)
    available_slots = booking_service.get_available_slots(master_id, date)

    return {
        "date": date.isoformat(),
        "master_id": master_id,
        "available_slots": available_slots
    }


@app.post("/public/booking", status_code=status.HTTP_201_CREATED)
async def create_public_booking(data: CreateBookingRequest, db: Session = Depends(get_db)):
    """
    Create a new booking (public endpoint).
    Sends WhatsApp confirmation.
    """
    tenant = db.query(Tenant).filter(Tenant.subdomain == data.subdomain).first()

    if not tenant or tenant.status not in [TenantStatus.ACTIVE, TenantStatus.TRIAL]:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Business not found or inactive"
        )

    # Get or create client
    client = db.query(Client).filter(Client.phone == data.client_phone).first()
    if not client:
        client = Client(
            phone=data.client_phone,
            full_name=data.client_name
        )
        db.add(client)
        db.flush()

    # Get service
    service = db.query(Service).filter(
        Service.id == data.service_id,
        Service.tenant_id == tenant.id
    ).first()

    if not service:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Service not found"
        )

    # Check if master provides this service
    master_service = db.query(MasterService).filter(
        MasterService.master_id == data.master_id,
        MasterService.service_id == data.service_id
    ).first()

    if not master_service:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Master does not provide this service"
        )

    # Check availability
    booking_service = BookingService(db)
    if not booking_service.is_slot_available(data.master_id, data.booking_date, service.duration_minutes):
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="Time slot not available"
        )

    try:
        # Create booking
        booking = Booking(
            tenant_id=tenant.id,
            client_id=client.id,
            master_id=data.master_id,
            service_id=data.service_id,
            booking_date=data.booking_date,
            duration_minutes=service.duration_minutes,
            price=service.price,
            status=BookingStatus.CONFIRMED,
            client_notes=data.notes
        )
        db.add(booking)
        db.commit()
        db.refresh(booking)

        logger.info(f"Booking created: ID={booking.id}")

        # Send WhatsApp confirmation
        if settings.WHATSAPP_ENABLED:
            try:
                async with httpx.AsyncClient() as client_http:
                    await client_http.post(
                        f"{WHATSAPP_SERVICE_URL}/send-message",
                        json={
                            "phone": data.client_phone,
                            "message": f"✅ Бронирование подтверждено!\n\n"
                                       f"Бизнес: {tenant.business_name}\n"
                                       f"Услуга: {service.name}\n"
                                       f"Дата: {data.booking_date.strftime('%d.%m.%Y %H:%M')}\n"
                                       f"Цена: {float(service.price)} ₸\n\n"
                                       f"Спасибо за ваш выбор!"
                        },
                        timeout=5.0
                    )
            except Exception as e:
                logger.error(f"Failed to send WhatsApp message: {e}")

        return {
            "message": "Booking created successfully",
            "booking_id": booking.id,
            "booking_date": booking.booking_date.isoformat(),
            "status": booking.status.value
        }

    except Exception as e:
        db.rollback()
        logger.error(f"Booking creation failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Booking creation failed"
        )


@app.get("/bookings")
async def get_bookings(
    user_id: int = Query(...),
    role: str = Query(...),
    tenant_id: Optional[int] = Query(None),
    date: Optional[date] = Query(None),
    status: Optional[str] = Query(None),
    db: Session = Depends(get_db)
):
    """
    Get bookings filtered by user role.
    """
    query = db.query(Booking)

    # Filter based on role
    if role == "OWNER" and tenant_id:
        query = query.filter(Booking.tenant_id == tenant_id)
    elif role == "MASTER":
        # Get master record for this user
        master = db.query(Master).filter(Master.user_id == user_id).first()
        if master:
            query = query.filter(Booking.master_id == master.id)
        else:
            return {"bookings": []}

    if date:
        start_of_day = datetime.combine(date, time.min)
        end_of_day = datetime.combine(date, time.max)
        query = query.filter(
            Booking.booking_date >= start_of_day,
            Booking.booking_date <= end_of_day
        )

    if status:
        query = query.filter(Booking.status == status)

    bookings = query.order_by(Booking.booking_date.desc()).all()

    return {
        "bookings": [
            {
                "id": b.id,
                "booking_date": b.booking_date.isoformat(),
                "status": b.status.value,
                "client_name": b.client.full_name if b.client else None,
                "client_phone": b.client.phone if b.client else None,
                "master_name": b.master.full_name if b.master else None,
                "price": float(b.price)
            }
            for b in bookings
        ]
    }


@app.delete("/booking/{booking_id}")
async def cancel_booking(
    booking_id: int,
    user_id: int = Query(...),
    role: str = Query(...),
    db: Session = Depends(get_db)
):
    """
    Cancel booking.
    Sends WhatsApp notification.
    """
    booking = db.query(Booking).filter(Booking.id == booking_id).first()

    if not booking:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Booking not found"
        )

    # Update status
    booking.status = BookingStatus.CANCELLED
    db.commit()

    # Send WhatsApp notification
    if settings.WHATSAPP_ENABLED and booking.client:
        try:
            async with httpx.AsyncClient() as client:
                await client.post(
                    f"{WHATSAPP_SERVICE_URL}/send-message",
                    json={
                        "phone": booking.client.phone,
                        "message": f"❌ Ваше бронирование отменено\n\n"
                                   f"Дата: {booking.booking_date.strftime('%d.%m.%Y %H:%M')}\n\n"
                                   f"Для новой записи свяжитесь с нами."
                    },
                    timeout=5.0
                )
        except Exception as e:
            logger.error(f"Failed to send WhatsApp cancellation: {e}")

    return {"message": "Booking cancelled successfully"}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host=settings.BOOKING_SERVICE_HOST if hasattr(settings, 'BOOKING_SERVICE_HOST') else "0.0.0.0",
        port=settings.BOOKING_SERVICE_PORT if hasattr(settings, 'BOOKING_SERVICE_PORT') else 8002,
        reload=settings.DEBUG
    )
