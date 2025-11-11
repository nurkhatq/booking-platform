#!/usr/bin/env python3
"""
Database initialization script.
Creates all tables and optionally seeds initial data.
"""
import sys
import os

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from shared.database import engine, Base, SessionLocal
from shared.models import User, Tenant, Location, UserRole, TenantStatus
from shared.auth import get_password_hash
from datetime import datetime, timedelta
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def create_tables():
    """Create all tables."""
    logger.info("Creating database tables...")
    Base.metadata.create_all(bind=engine)
    logger.info("Tables created successfully!")


def seed_data():
    """Seed initial data (optional)."""
    logger.info("Seeding initial data...")

    db = SessionLocal()

    try:
        # Check if super admin already exists
        existing_admin = db.query(User).filter(
            User.role == UserRole.SUPER_ADMIN
        ).first()

        if existing_admin:
            logger.info("Super admin already exists, skipping seed data")
            return

        # Create super admin user
        admin = User(
            email="admin@jazyl.tech",
            phone="+77771234567",
            password_hash=get_password_hash("admin123"),
            full_name="Super Admin",
            role=UserRole.SUPER_ADMIN,
            is_active=True,
            tenant_id=None
        )
        db.add(admin)
        db.commit()

        logger.info("Super admin created:")
        logger.info("  Email: admin@jazyl.tech")
        logger.info("  Password: admin123")
        logger.info("  ⚠️  CHANGE THIS PASSWORD IN PRODUCTION!")

        # Optionally create a demo tenant
        demo_tenant = Tenant(
            subdomain="demo",
            business_name="Demo Salon",
            phone="+77771234567",
            email="demo@jazyl.tech",
            description="Demo business for testing",
            status=TenantStatus.ACTIVE,
            trial_end_date=datetime.utcnow() + timedelta(days=365)
        )
        db.add(demo_tenant)
        db.flush()

        # Create demo location
        demo_location = Location(
            tenant_id=demo_tenant.id,
            name="Demo Salon - Main Branch",
            address="Almaty, Kazakhstan",
            phone="+77771234567",
            is_main=True
        )
        db.add(demo_location)

        # Create demo owner
        demo_owner = User(
            tenant_id=demo_tenant.id,
            email="owner@demo.jazyl.tech",
            phone="+77779876543",
            password_hash=get_password_hash("demo123"),
            full_name="Demo Owner",
            role=UserRole.OWNER,
            is_active=True
        )
        db.add(demo_owner)

        db.commit()

        logger.info("\nDemo tenant created:")
        logger.info("  Subdomain: demo")
        logger.info("  Owner email: owner@demo.jazyl.tech")
        logger.info("  Owner password: demo123")

    except Exception as e:
        db.rollback()
        logger.error(f"Error seeding data: {e}")
        raise
    finally:
        db.close()


if __name__ == "__main__":
    try:
        create_tables()
        seed_data()
        logger.info("\n✅ Database initialization completed successfully!")

    except Exception as e:
        logger.error(f"\n❌ Database initialization failed: {e}")
        sys.exit(1)
