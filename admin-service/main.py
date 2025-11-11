from fastapi import FastAPI, HTTPException, status, Depends
from sqlalchemy.orm import Session
from sqlalchemy import func
from datetime import datetime, timedelta
import logging

from shared.config import settings
from shared.database import get_db, check_db_connection
from shared.models import Tenant, Booking, User, TenantStatus

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="Admin Service",
    description="Platform administration service",
    version="2.0.0"
)


@app.on_event("startup")
async def startup_event():
    """Initialize on startup."""
    logger.info("Starting Admin Service...")
    if check_db_connection():
        logger.info("Database connection successful")


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "admin-service"
    }


@app.get("/tenants/pending")
async def get_pending_tenants(db: Session = Depends(get_db)):
    """
    Get all pending tenant applications.
    """
    tenants = db.query(Tenant).filter(
        Tenant.status == TenantStatus.PENDING
    ).order_by(Tenant.created_at.desc()).all()

    return {
        "tenants": [
            {
                "id": t.id,
                "business_name": t.business_name,
                "subdomain": t.subdomain,
                "phone": t.phone,
                "email": t.email,
                "created_at": t.created_at.isoformat(),
                "status": t.status.value
            }
            for t in tenants
        ]
    }


@app.put("/tenant/{tenant_id}/approve")
async def approve_tenant(tenant_id: int, db: Session = Depends(get_db)):
    """
    Approve tenant application.
    """
    tenant = db.query(Tenant).filter(Tenant.id == tenant_id).first()

    if not tenant:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Tenant not found"
        )

    tenant.status = TenantStatus.ACTIVE
    db.commit()

    logger.info(f"Tenant approved: {tenant.subdomain}")

    return {
        "message": "Tenant approved",
        "tenant_id": tenant.id,
        "status": tenant.status.value
    }


@app.put("/tenant/{tenant_id}/reject")
async def reject_tenant(tenant_id: int, db: Session = Depends(get_db)):
    """
    Reject tenant application.
    """
    tenant = db.query(Tenant).filter(Tenant.id == tenant_id).first()

    if not tenant:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Tenant not found"
        )

    tenant.status = TenantStatus.REJECTED
    db.commit()

    logger.info(f"Tenant rejected: {tenant.subdomain}")

    return {
        "message": "Tenant rejected",
        "tenant_id": tenant.id,
        "status": tenant.status.value
    }


@app.get("/statistics")
async def get_statistics(db: Session = Depends(get_db)):
    """
    Get platform statistics.
    """
    # Count tenants by status
    total_tenants = db.query(func.count(Tenant.id)).scalar()
    active_tenants = db.query(func.count(Tenant.id)).filter(
        Tenant.status == TenantStatus.ACTIVE
    ).scalar()
    trial_tenants = db.query(func.count(Tenant.id)).filter(
        Tenant.status == TenantStatus.TRIAL
    ).scalar()
    pending_tenants = db.query(func.count(Tenant.id)).filter(
        Tenant.status == TenantStatus.PENDING
    ).scalar()

    # Count bookings
    total_bookings = db.query(func.count(Booking.id)).scalar()

    # Count users
    total_users = db.query(func.count(User.id)).scalar()

    # Recent bookings (last 30 days)
    thirty_days_ago = datetime.utcnow() - timedelta(days=30)
    recent_bookings = db.query(func.count(Booking.id)).filter(
        Booking.created_at >= thirty_days_ago
    ).scalar()

    return {
        "tenants": {
            "total": total_tenants,
            "active": active_tenants,
            "trial": trial_tenants,
            "pending": pending_tenants
        },
        "bookings": {
            "total": total_bookings,
            "last_30_days": recent_bookings
        },
        "users": {
            "total": total_users
        }
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host=settings.ADMIN_SERVICE_HOST if hasattr(settings, 'ADMIN_SERVICE_HOST') else "0.0.0.0",
        port=settings.ADMIN_SERVICE_PORT if hasattr(settings, 'ADMIN_SERVICE_PORT') else 8005,
        reload=settings.DEBUG
    )
